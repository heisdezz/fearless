package commands

// Cancel terminates an in-flight request execution.
// If RequestUID is specified, that specific execution is cancelled.
// If ItemUID is specified, any active execution for that item is cancelled.
// Returns true if an active request was found and cancelled, false otherwise.
func (s *RequestService) Cancel(cmd CancelRequestCmd) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cmd.RequestUID != "" {
		if req, ok := s.active[cmd.RequestUID]; ok {
			req.cancel()
			return true
		}
	}

	if cmd.ItemUID != "" {
		if reqUID, ok := s.byItem[cmd.ItemUID]; ok {
			if req, exists := s.active[reqUID]; exists {
				req.cancel()
				return true
			}
		}
	}

	return false
}

// InFlight reports whether an execution is currently in progress for the item.
func (s *RequestService) InFlight(itemUID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.byItem[itemUID]
	return exists
}

// ActiveCount returns the number of requests currently in flight.
func (s *RequestService) ActiveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.active)
}
