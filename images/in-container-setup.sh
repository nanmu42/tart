# Use apt mirror
cat <<EOF > /etc/apt/sources.list
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-updates main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-backports main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-security main restricted universe multiverse
EOF

# necessary packages, including systemd
packages="ca-certificates udev systemd-sysv iproute2 curl tzdata zip"
DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y --no-install-recommends $packages

# Disable resolved and ntpd
rm -f /etc/systemd/system/multi-user.target.wants/systemd-resolved.service
rm -f /etc/systemd/system/dbus-org.freedesktop.resolve1.service
rm -f /etc/systemd/system/sysinit.target.wants/systemd-timesyncd.service

# DNS
cat <<EOF > /etc/resolv.conf
nameserver 223.5.5.5
nameserver 223.6.6.6
EOF

# Auto-login
# The serial getty service hooks up the login prompt to the kernel console at
# ttyS0 (where Firecracker connects its serial console).
# We'll set it up for autologin to avoid the login prompt.
passwd -d root
mkdir "/etc/systemd/system/serial-getty@ttyS0.service.d/"
cat <<EOF > "/etc/systemd/system/serial-getty@ttyS0.service.d/autologin.conf"
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin root -o '-p -- \\u' --keep-baud 115200,38400,9600 %I $TERM
EOF

# Install Go
curl -Lo go_installer https://get.golang.org/linux && chmod +x go_installer && ./go_installer && rm go_installer
mv /root/.go /usr/local/go # go_installer has weird default installation location
echo 'PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin"' > /etc/environment