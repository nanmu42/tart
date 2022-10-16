# Firecracker RootFS and Kernel

## Kernel

To build [a Kernel supported by Firecracker](https://github.com/firecracker-microvm/firecracker/blob/main/docs/kernel-policy.md), follow the [guide](https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md#manual-compilation) and [configurations](https://github.com/firecracker-microvm/firecracker/blob/main/resources/guest_configs).

Use `make vmlinux -j$(nproc)` can accelerate the building process.

## RootFS

```bash
bash build-jammy.sh
```

## Boot a VM

After configure a TAP device following the [official doc](https://github.com/firecracker-microvm/firecracker/blob/main/docs/network-setup.md), run:

```bash
firectl --kernel ./vmlinux-5.10.bin --root-drive ./jammy.rootfs.ext4 --kernel-opts 'ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules ip=172.18.0.2::172.18.0.1:255.255.255.0::eth0:off' --tap-device 'tap0/AA:FC:00:00:00:01'
```

Note the `ip` parameter may need be tweak per your local network configuration.

## References

* https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md
* https://jvns.ca/blog/2021/01/23/firecracker--start-a-vm-in-less-than-a-second/