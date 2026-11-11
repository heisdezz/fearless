#!/usr/bin/env python3
"""Qualify bare `index.js:42` citations to full src-relative paths using each
document's declared base directory, so citations resolve without prose context.

Run from the repo root:  python3 ui-map/_tools/qualify.py
"""

import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[2]
MAP = ROOT / "ui-map"
SRC = ROOT / "bruno/packages/bruno-app/src"

# The component directory each screen/component file is about. A bare
# `index.js` / `StyledWrapper.js` in that file refers to this component.
BASES = {
    "screens/app-shell.md": ["components/AppTitleBar", "components/StatusBar"],
    "screens/cookies.md": ["components/Cookies"],
    "screens/global-search.md": ["components/GlobalSearchModal"],
    "screens/notifications.md": ["components/Notifications/NotificationsModal"],
    "screens/welcome.md": ["components/WelcomeModal"],
    "screens/about.md": ["components/Bruno"],
    "screens/environments.md": [
        "components/Environments",
        "components/EnvironmentVariablesTable",
    ],
    "screens/collection-settings.md": ["components/CollectionSettings"],
    "screens/folder-settings.md": ["components/FolderSettings"],
    "screens/preferences.md": ["components/Preferences"],
    "screens/sidebar.md": ["components/Sidebar"],
    "screens/sidebar-dialogs.md": ["components/Sidebar"],
    "screens/request-tabs.md": ["components/RequestTabs"],
    "screens/request-pane.md": ["components/RequestPane"],
    "screens/request-params.md": ["components/RequestPane"],
    "screens/request-body.md": ["components/RequestPane/RequestBody"],
    "screens/request-auth.md": ["components/RequestPane/Auth"],
    "screens/request-subtabs.md": ["components/RequestPane"],
    "screens/request-docs.md": ["components/Documentation"],
    "screens/response-pane.md": ["components/ResponsePane"],
    "screens/response-viewer.md": ["components/ResponsePane"],
    "screens/runner.md": ["components/RunnerResults"],
    "components/button.md": ["ui/Button"],
    "components/method-badge.md": ["ui/MethodBadge"],
    "components/spinner.md": ["components/Spinner"],
    "components/toggle.md": ["ui/Toggle"],
    "components/tabs.md": ["ui/Tabs"],
    "components/text-input.md": ["ui/TextInput"],
    "components/select.md": ["ui/Select"],
    "components/status-badge.md": ["ui/StatusBadge"],
}

# Only bare filenames (no slash) are rewritten; paths already containing a
# directory are left alone.
BARE = re.compile(r"(?<![A-Za-z0-9_./-])((?:index|StyledWrapper)\.(?:jsx?|tsx?)):(\d+)")


def main():
    changed = 0
    for rel, bases in BASES.items():
        p = MAP / rel
        if not p.exists():
            print("skip missing", rel)
            continue
        txt = p.read_text()

        def repl(m):
            nonlocal changed
            name = m.group(1)
            for b in bases:
                if (SRC / b / name).exists():
                    changed += 1
                    return f"{b}/{name}:{m.group(2)}"
            return m.group(0)

        new = BARE.sub(repl, txt)
        if new != txt:
            p.write_text(new)
    print("qualified", changed, "bare citations")


if __name__ == "__main__":
    main()
