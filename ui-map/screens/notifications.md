# Screen — notifications

Mode: **Operate**. A side panel/modal opened from the status-bar bell.

Source: `components/Notifications/` (`index.js`, `NotificationsModal/`:
`index.js`, `NotificationList.js`, `NotificationDetail.js`, `NotificationTabs.js`).
Test hook: bell has `aria-label="Check all Notifications"` (`components/Notifications/NotificationsModal/index.js:13`).

---

## Layout tree

```text
┌────────────────────────────────────────────────────────────────┐
│ Notifications  [ All ] [ Unread ]        (menu ⋮)              │
├──────────────────────────┬─────────────────────────────────────┤
│ • Update available …      │  <detail of selected notification>  │
│ • CI run passed …         │                                     │
│ (list, resizable)         │                                     │
└──────────────────────────┴─────────────────────────────────────┘
```

- Two-column panel: list | detail, split by a draggable divider
  (`aria-label="Resize sidebar"`, `NotificationsModal/index.js:74`).
- Tabs: notification filters (`NotificationTabs.js`), menu trigger
  `aria-label="Notifications menu"`.

## States & copy (verbatim)

| State | Copy |
| --- | --- |
| Nothing at all | **`You are all caught up!`** (`components/Notifications/NotificationsModal/index.js:78`) |
| Empty tab | **`No notifications to show.`** (`NotificationList.js:30`) |
| No selection | **`Select a notification to read more.`** (`NotificationDetail.js:28`) |

## Interactions

- Click a list item → detail pane.
- Drag the divider to resize the list.
- Bell badge reflects unread count (see `StatusBar/index.js`).
