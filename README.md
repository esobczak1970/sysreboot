# sysreboot - Enhanced Reboot Tool

`sysreboot` is a smart, enhanced reboot tool designed to provide a safer, more efficient, and user-friendly way to manage system restarts and shutdowns. Developed with the aim of overcoming the limitations and complexities associated with traditional shutdown and reboot scripts, `sysreboot` offers a streamlined command-line experience.

## Features

- **Safer Operations**: Requires confirmation flags to execute shutdown operations, preventing accidental system halts.
- **Efficiency**: Executes complex operations like scheduled reboots, delayed actions, and custom broadcast messages with simple command-line flags.
- **Convenience**: Defaults to reboot, reducing the risk of accidental shutdowns when reboot is intended.
- **Versatility**: Supports scheduling actions at a specific time, adding a delay before an action, and sending messages to all users.
- **Confirmation Prompts**: Ensures deliberate actions by prompting the user for confirmation if required.
- **Verbose Output**: Provides detailed logging for actions taken, which can be enabled with a verbose flag.

## Benefits Over Traditional Scripts

- **Intuitive Default Behavior**: The default action is to reboot, which aligns with the most common use case, eliminating the risk of unintentional shutdowns.
- **Confirmation and Safety**: The need for an explicit confirmation for shutdowns adds a layer of safety, ensuring that critical operations are intentional.
- **User Messaging**: Ability to send messages to all users before an action provides clarity and communication, minimizing disruption.
- **Scheduling Made Simple**: Instead of writing complex scripts, schedule system reboots or shutdowns with simple flags.

## Usage Examples

### Rebooting the System (Default Action)

- **Long Form**: `sysreboot --reboot`
- **Short Form**: `sysreboot -r`

### Rebooting with a Delay

- **Long Form**: `sysreboot --reboot --delay 10 --message "Rebooting in 10 minutes"`
- **Short Form**: `sysreboot -r -d 10 -m "Rebooting in 10 minutes"`

### Powering Off with Confirmation

- **Long Form**: `sysreboot --poweroff --confirm`
- **Short Form**: `sysreboot -p -c`

### Scheduling a Reboot at a Specific Time

- **Long Form**: `sysreboot --reboot --time "23:30" --message "Scheduled reboot at 23:30"`
- **Short Form**: `sysreboot -r -t "23:30" -m "Scheduled reboot at 23:30"`

### Verbose Logging

- **Long Form**: `sysreboot --verbose`
- **Short Form**: `sysreboot -vb`

By integrating these features into `sysreboot`, the tool not only streamlines the process but also adds a layer of confirmation that prevents accidental system halts or shutdowns, thereby promoting a safer environment for system administrators and users alike.

## Getting Started

To get started with `sysreboot`, clone the repository and build the tool with Go:

```sh
git clone https://github.com/esobczak1970/sysreboot.git
cd sysreboot
go build
```

## Contributing

Contributions to `sysreboot` are welcome. Please feel free to submit issues, fork the repository, and send pull requests!

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

---
