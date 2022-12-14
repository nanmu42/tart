set -exo pipefail

# Use apt mirror
cat <<EOF > /etc/apt/sources.list
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-updates main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-backports main restricted universe multiverse
deb http://mirrors.bfsu.edu.cn/ubuntu/ jammy-security main restricted universe multiverse
EOF

# necessary packages, including systemd and ssh server
packages="ca-certificates udev systemd-sysv iproute2 curl tzdata zip openssh-server git build-essential"
DEBIAN_FRONTEND=noninteractive apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y $packages < /dev/null # by default apt-get openssh-server reads from Stdin, stops script execution.
rm -rf /var/lib/apt/lists/* # clear APT cache

# ssh login method
echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config
mkdir -p /root/.ssh
echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCtJ4YxqZ6O6pUuaJpXqVMyTyRIfsU20sqS8gKAlx+xyRjb79C1UQpX60wpV6fH3s1wS2InItQPnrYItqofkQAgmn6POCzp1eaBMVWOw1Ke4LAC5KUONc/0nzoTzDR/5cle9aNd3yYDGTlm8ZXDx6xOVIrI9ymC2E5T3rQJ8aB22iEBMMoOAdUnhCB7dAtn1cCCLBAw9rqswJGr4ngYiID5eYfzeladGp9KqyFA8aAluL3JW12TAwM10ENG03wk8laXCfa7ECBC2OWwlLnoJI2Wnbfw67Wobt6yrcD7KJ2xpL9YBj8g2XPu4B29rqde3DFNmpLV8jtlqI9rE4S2igYX ci@tart-host' > /root/.ssh/authorized_keys

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