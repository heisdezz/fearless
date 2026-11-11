package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/collection"
	"fearless/internal/core/engine"
	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// RunOptions configures a headless collection execution run.
type RunOptions struct {
	Path      string
	Env       string
	EnvVars   map[string]any
	Recursive bool
	Bail      bool
	Format    string // "text" or "json"
	Stdout    io.Writer
	Stderr    io.Writer
}

// RunSummary summarizes the results of running one or more requests.
type RunSummary struct {
	Total    int                    `json:"total"`
	Passed   int                    `json:"passed"`
	Failed   int                    `json:"failed"`
	Duration time.Duration          `json:"duration"`
	Results  []RequestResultSummary `json:"results"`
}

// RequestResultSummary records the result of a single request.
type RequestResultSummary struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	URL        string        `json:"url"`
	Status     int           `json:"status"`
	StatusText string        `json:"statusText"`
	Duration   time.Duration `json:"duration"`
	Passed     bool          `json:"passed"`
	Error      string        `json:"error,omitempty"`
}

// ExecuteRun executes requests according to RunOptions and returns a summary.
func ExecuteRun(ctx context.Context, opts RunOptions) (RunSummary, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	start := time.Now()
	col, items, err := loadTarget(opts.Path, opts.Recursive)
	if err != nil {
		return RunSummary{}, err
	}

	// Environment resolution: if no env specified, check collection defaults
	envName := opts.Env
	if envName == "" && len(col.Environments) > 0 {
		// If an active environment was set or preset, prefer it
		for _, e := range col.Environments {
			if strings.EqualFold(e.Name, "default") || strings.EqualFold(e.Name, "dev") {
				envName = e.Name
				break
			}
		}
	}

	b := bus.New()
	defer b.Close()

	svc := commands.NewRequestService(engine.New(), nil, b, nil, nil)
	summary := RunSummary{}

	if opts.Format != "json" {
		fmt.Fprintf(opts.Stdout, "Running collection: %s (%d requests)\n", col.Name, len(items))
		if envName != "" {
			fmt.Fprintf(opts.Stdout, "Environment: %s\n", envName)
		}
		fmt.Fprintln(opts.Stdout, strings.Repeat("-", 60))
	}

	for _, it := range items {
		if ctx.Err() != nil {
			break
		}

		itemReq := it.Request
		if it.Draft != nil {
			itemReq = it.Draft
		}
		if itemReq == nil {
			continue
		}

		reqSummary := RequestResultSummary{
			Name:   it.Name,
			Path:   it.Path,
			Method: itemReq.Method,
			URL:    itemReq.URL,
		}
		summary.Total++

		cmd := commands.SendRequestCmd{
			Collection:  col,
			Item:        it,
			Env:         envName,
			RuntimeVars: opts.EnvVars,
		}

		res, sendErr := svc.SendSync(ctx, cmd)
		if sendErr != nil {
			reqSummary.Passed = false
			reqSummary.Error = sendErr.Error()
			summary.Failed++
		} else if res != nil {
			reqSummary.Status = res.Status
			reqSummary.StatusText = res.StatusText
			reqSummary.Duration = res.Duration

			if res.Failed() || res.Status >= 400 {
				reqSummary.Passed = false
				reqSummary.Error = res.Err
				if reqSummary.Error == "" && res.Status >= 400 {
					reqSummary.Error = fmt.Sprintf("HTTP %d %s", res.Status, res.StatusText)
				}
				summary.Failed++
			} else {
				reqSummary.Passed = true
				summary.Passed++
			}
		}

		summary.Results = append(summary.Results, reqSummary)

		if opts.Format != "json" {
			if reqSummary.Passed {
				fmt.Fprintf(opts.Stdout, "✓ %d %s (%dms) - %s\n",
					reqSummary.Status, reqSummary.StatusText, reqSummary.Duration.Milliseconds(), reqSummary.Name)
			} else {
				fmt.Fprintf(opts.Stdout, "✗ %d %s (%dms) - %s\n",
					reqSummary.Status, reqSummary.StatusText, reqSummary.Duration.Milliseconds(), reqSummary.Name)
				if reqSummary.Error != "" {
					fmt.Fprintf(opts.Stdout, "  Error: %s\n", reqSummary.Error)
				}
			}
		}

		if opts.Bail && !reqSummary.Passed {
			break
		}
	}

	summary.Duration = time.Since(start)

	if opts.Format == "json" {
		enc := json.NewEncoder(opts.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(summary)
	} else {
		fmt.Fprintln(opts.Stdout, strings.Repeat("-", 60))
		fmt.Fprintf(opts.Stdout, "Ran %d requests (%d passed, %d failed) in %v\n",
			summary.Total, summary.Passed, summary.Failed, summary.Duration.Round(time.Millisecond))
	}

	return summary, nil
}

// loadTarget loads a collection or individual file into a model.Collection and runnable items.
func loadTarget(path string, recursive bool) (*model.Collection, []*model.Item, error) {
	if path == "" {
		path = "."
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("stat target path %s: %w", absPath, err)
	}

	if !info.IsDir() {
		// Single request file
		data, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return nil, nil, fmt.Errorf("read request file %s: %w", absPath, rerr)
		}

		codec, cerr := format.For(model.FormatBru)
		if cerr != nil {
			return nil, nil, cerr
		}
		if strings.HasSuffix(absPath, ".yml") || strings.HasSuffix(absPath, ".yaml") {
			if ycodec, ok := format.Get(model.FormatYML); ok {
				codec = ycodec
			}
		}

		req, perr := codec.ParseRequest(data)
		if perr != nil {
			return nil, nil, fmt.Errorf("parse request %s: %w", absPath, perr)
		}

		itemName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
		it := &model.Item{
			UID:      "single-item",
			Type:     model.ItemHTTPRequest,
			Name:     itemName,
			Path:     absPath,
			Filename: filepath.Base(absPath),
			Request:  req,
		}

		col := &model.Collection{
			UID:   "single-col",
			Name:  itemName,
			Path:  filepath.Dir(absPath),
			Items: []*model.Item{it},
		}
		return col, []*model.Item{it}, nil
	}

	// Collection directory
	col, lerr := collection.Load(absPath)
	if lerr != nil {
		return nil, nil, fmt.Errorf("load collection from %s: %w", absPath, lerr)
	}

	var items []*model.Item
	if recursive {
		_ = col.Walk(func(it *model.Item) error {
			if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
				items = append(items, it)
			}
			return nil
		})
	} else {
		for _, it := range col.Items {
			if it.Type == model.ItemHTTPRequest || (it.Type == "" && it.Request != nil) {
				items = append(items, it)
			}
		}
	}

	return col, items, nil
}
