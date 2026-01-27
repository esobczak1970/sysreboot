# sysreboot - Enhanced Reboot and Shutdown Utility

`sysreboot` is an intuitive command-line utility designed to manage system reboots and shutdowns with enhanced safety and user-friendly features. It simplifies the execution of restarts and shutdowns, offering a significant improvement over traditional commands.

## Features

- **Safe Defaults**: Reboot with a delay unless specified with `-d 0` for an immediate reboot.
- **Explicit Confirmation**: Shutdown operations require confirmation to avoid accidental power-offs.
- **Simplified Scheduling**: Easily schedule reboots or shutdowns with a specific time or after a delay.
- **Communication Clarity**: Send custom broadcast messages before system actions.
- **Detailed Feedback**: Opt for verbose output to monitor system actions closely.

## Comparative Advantages

Here's how `sysreboot` simplifies system management compared to traditional commands:

| Action | sysreboot Command | Traditional Command(s) |
| ------ | ----------------- | ---------------------- |
| Immediate Reboot | `sysreboot -d 0` | `reboot` or `shutdown -r now` |
| Confirmed Shutdown | `sysreboot --shutdown --confirm` | `shutdown -h now "Confirm shutdown? y/n:" && read confirmation` |
| Scheduled Reboot | `sysreboot --time "23:30" -m "Scheduled reboot at 23:30"` | `echo "shutdown -r now" \| at 23:30` |
| Delayed Reboot | `sysreboot -d 5 -m "Rebooting in 5 minutes"` | `sleep 300 && shutdown -r now` |
| Halt with Delay | `sysreboot --halt` | `shutdown -H +1 "Halt in 1 minute"` |
| Verbose Logging | `sysreboot -vb` | `shutdown -v -r now` |

## Additional Features

- **Countdown**: Utilize `--countdown` (`-co`) to display a countdown to the action.
- **Background Execution**: Use `--background` (`-b`) to send the command to run in the background.

## Getting Started

To start using `sysreboot`, follow these steps:

```sh
git clone https://github.com/esobczak1970/sysreboot.git
cd sysreboot
go build
```

## Contributing

Your contributions can help improve `sysreboot`. Feel free to report issues, fork the repository, and submit pull requests.

## License

`sysreboot` is made available under the MIT License. Refer to [LICENSE.md](LICENSE.md) for details.

---
