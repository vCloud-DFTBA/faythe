# Note

## Time durations

Time durations are specified as a number, followed immediately by one of the following units:

- `ms` - milliseconds
- `s` - seconds
- `m` - minutes
- `h` - hours
- `d` - days
- `w` - weeks
- `y` - years

For example, to set `autoscaling.interval` to 5 minutes: `interval: 5m`.

Some invalid format examples:

- `1h30m`
- `5M`
- ...
