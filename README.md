# ZFSSnap

ZFSSnap is an application that automatically takes and manages scheduled daily, weekly, and monthly snapshots of ZFS
filesystems.

## Configuration

ZFSSnap is configured through a basic YAML config file. For example, this config file will take and rotate up to 1
daily, 2 weekly, and 2 monthly snapshots for the filesystem named 'tank', and only 1 monthly snapshot for the filesystem
named 'dozer'.

```yaml
- name: tank
  daily: 1
  weekly: 2
  monthly: 2
- name: dozer
  daily: 0
  weekly: 0
  monthly: 1
```

The properties are:

- `name`: The name of the filesystem to snapshot
- `daily`: The maximum number of daily snapshots to retain. Can be 0.
- `weekly`: The maximum number of weekly snapshots to retain. Can be 0.
- `monthly`: The maximum number of monthly snapshots to retain. Can be 0.

## Snapshots

All snapshots follow the nameing convention of: `<filesystem>@auto_<type>_<index>`. Where `type` is one of `daily`,
`weekly`, or `monthly`.

The index describes the date for the given snapshot relative to the type. Daily snapshots will have the full date,
weekly snapshots will have the year and week number, and monthly snapshots will have the year and month number.

ZFSSnap ignores snapshots that don't match the following pattern `auto_(daily|weekly|monthly)_[0-9]+$`.
