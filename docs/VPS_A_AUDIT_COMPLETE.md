# Radio VPS Audit Results — 72.60.181.89 (2026-03-16)
#
# IMPORTANT: This audit was performed on the RADIO VPS (72.60.181.89),
# which runs radio, market (pwl-market), more (pwl-more), and stream.
# This is NOT an audit of the NiteOS VPS (31.97.126.86).
#
# Key findings (summarised):
# - Ubuntu 22.04.5 LTS, Docker 29.2.1
# - 8 containers running: pwl-market, pwl-more, radio-rtmp, radio-web, radio-autodj + support
# - nginx on :80/:443, no Traefik, no NiteOS stack
# - /opt/niteos: absent (correct — NiteOS does not run here)
# - service-1: absent (correct — never deployed to this machine)
# - pwl-market-backup.timer: active
# - 35 GB disk free
#
# Raw output follows.

# OS version and kernel
lsb_release -a
uname -r

# CPU / RAM
nproc
free -h

# Disk usage
df -h

# Block devices
lsblk
No LSB modules are available.
Distributor ID: Ubuntu
Description:    Ubuntu 22.04.5 LTS
Release:        22.04
Codename:       jammy
5.15.0-164-generic
1
               total        used        free      shared  buff/cache   available
Mem:           3.8Gi       892Mi       641Mi        38Mi       2.3Gi       2.6Gi
Swap:             0B          0B          0B
Filesystem      Size  Used Avail Use% Mounted on
tmpfs           392M  1.7M  390M   1% /run
/dev/sda1        49G   14G   35G  29% /
tmpfs           2.0G     0  2.0G   0% /dev/shm
tmpfs           5.0M     0  5.0M   0% /run/lock
/dev/sda15      105M  6.1M   99M   6% /boot/efi
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/46bd6275f081c237e7f8f1a6b3e79565a16f96c0beb5720ba8927d1cba45b808
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/f0d8bc093c989f2c3bcbfe2e77d4df4d75a209a2a9086effc8ff26d90542aa60
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/6b1986bc91e42a9199d338dc00df7152e67e6e78c1ae7e27eea126d15f69929a
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/a5db5cb467ae557c719c746243b2e7e05cb375122f9d1088627633a71bc033e4
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/da10f18bbaae82d774e9253046b835d38c6dd7de5b7f26e421d3f0d850f50c51
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/4ba77071e15e797eee7ac1c6f675b1bb014e959706ce5c8e190cbb051960d00f
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/4f231893f8936b2008f2579d930acf192d76b4e1759d8cd1b1f111331454f4f6
overlay          49G   14G   35G  29% /var/lib/docker/rootfs/overlayfs/f836bf5525fb04c1d544cad0f8e9cfc67f2e7adb8d5deda67e7918b0aaaf8791
tmpfs           392M  4.0K  392M   1% /run/user/0
NAME    MAJ:MIN RM  SIZE RO TYPE MOUNTPOINTS
loop0     7:0    0 50.9M  1 loop /snap/snapd/25577
loop1     7:1    0 48.1M  1 loop /snap/snapd/25935
loop2     7:2    0 63.8M  1 loop /snap/core20/2686
loop4     7:4    0 63.8M  1 loop /snap/core20/2717
sda       8:0    0   50G  0 disk
├─sda1    8:1    0 49.9G  0 part /var/lib/docker/volumes/radijas-v2_hls-data/_data
│                                /var/lib/docker/volumes/radijas-v2_status-data/_data
│                                /
├─sda14   8:14   0    4M  0 part
└─sda15   8:15   0  106M  0 part /boot/efi
sr0      11:0    1    4M  0 rom
root@srv1178155:~#

Docker version 29.2.1, build a5c7197
Docker Compose version v5.0.2
psql: NOT INSTALLED
pg_lsclusters: not available
redis-cli: NOT INSTALLED
/usr/bin/python3
/usr/bin/certbot

linux-headers-5.15.0-171/jammy-updates,jammy-security,now 5.15.0-171.181 all [installed,auto-removable]
linux-headers-generic/jammy-security,now 5.15.0.171.160 amd64 [installed,upgradable to: 5.15.0.173.161]
linux-headers-virtual/jammy-security,now 5.15.0.171.160 amd64 [installed,upgradable to: 5.15.0.173.161]
linux-libc-dev/jammy-security,now 5.15.0-171.181 amd64 [installed,upgradable to: 5.15.0-173.183]
liquidsoap/jammy,now 2.0.2-1build2 amd64 [installed]
login/jammy-updates,jammy-security,now 1:4.8.1-2ubuntu2.2 amd64 [installed]
lsb-release/jammy,now 11.1.0ubuntu4 all [installed]
monarx-agent/jammy,now 4.3.10-master amd64 [installed,upgradable to: 4.3.11-master]
monarx-protect-autodetect/now 5.2.3-master amd64 [installed,upgradable to: 5.2.5-master]
monarx-protect/now 5.2.3-master amd64 [installed,upgradable to: 5.2.5-master]
nano/jammy-updates,jammy-security,now 6.2-1ubuntu0.1 amd64 [installed]
ncurses-base/jammy-updates,jammy-security,now 6.3-2ubuntu0.1 all [installed]
ncurses-bin/jammy-updates,jammy-security,now 6.3-2ubuntu0.1 amd64 [installed]
ncurses-term/jammy-updates,jammy-security,now 6.3-2ubuntu0.1 all [installed]
net-tools/jammy-updates,jammy-security,now 1.60+git20181103.0eebece-1ubuntu5.4 amd64 [installed]
nftables/now 1.0.2-1ubuntu3 amd64 [installed,upgradable to: 1.0.2-1ubuntu3.1]
nginx/jammy-updates,jammy-security,now 1.18.0-6ubuntu14.8 amd64 [installed]
openssh-server/jammy-updates,jammy-security,now 1:8.9p1-3ubuntu0.14 amd64 [installed]
openssh-sftp-server/jammy-updates,jammy-security,now 1:8.9p1-3ubuntu0.14 amd64 [installed]
python-babel-localedata/jammy,now 2.8.0+dfsg.1-7 all [installed]
python3-babel/jammy,now 2.8.0+dfsg.1-7 all [installed]
python3-certbot-nginx/jammy,now 1.21.0-1 all [installed]
python3-certifi/jammy,now 2020.6.20-1 all [installed]
python3-distutils/jammy-updates,jammy-security,now 3.10.8-1~22.04 all [installed]
python3-jinja2/jammy-updates,jammy-security,now 3.0.3-1ubuntu0.4 all [installed]
python3-json-pointer/jammy,now 2.0-0ubuntu1 all [installed]
python3-jsonpatch/jammy,now 1.32-2 all [installed]
python3-jsonschema/jammy,now 3.2.0-0ubuntu2 all [installed]
python3-lib2to3/jammy-updates,jammy-security,now 3.10.8-1~22.04 all [installed]
python3-markupsafe/jammy,now 2.0.1-2build1 amd64 [installed]
python3-pip/jammy-updates,jammy-security,now 22.0.2+dfsg-1ubuntu0.7 all [installed]
python3-pyrsistent/jammy,now 0.18.1-1build1 amd64 [installed]
python3-requests/jammy-updates,jammy-security,now 2.25.1+dfsg-2ubuntu0.3 all [installed]
python3-serial/jammy,now 3.5-1 all [installed]
python3-setuptools/jammy-updates,jammy-security,now 59.6.0-1.2ubuntu0.22.04.3 all [installed]
python3-tz/jammy-updates,now 2022.1-1ubuntu0.22.04.1 all [installed]
python3-urllib3/jammy-updates,jammy-security,now 1.26.5-1~exp1ubuntu0.6 all [installed]
python3-venv/jammy-updates,now 3.10.6-1~22.04.1 amd64 [installed]
python3/jammy-updates,now 3.10.6-1~22.04.1 amd64 [installed]
qemu-guest-agent/jammy-updates,jammy-security,now 1:6.2+dfsg-2ubuntu6.28 amd64 [installed]
sbsigntool/jammy,now 0.9.4-2ubuntu2 amd64 [installed]
secureboot-db/jammy,now 1.8 amd64 [installed]
shim-signed/jammy-updates,jammy-security,now 1.51.4+15.8-0ubuntu1 amd64 [installed]
software-properties-common/jammy-updates,now 0.99.22.9 all [installed]
sosreport/now 4.9.2-0ubuntu0~22.04.1 amd64 [installed,upgradable to: 4.10.2-0ubuntu0~22.04.1]
ssh-import-id/jammy,now 5.11-0ubuntu1 all [installed]
sysvinit-utils/jammy,now 3.01-1ubuntu1 amd64 [installed]
tcl8.6/jammy,now 8.6.12+dfsg-1build1 amd64 [installed]
tcl/jammy,now 8.6.11+1build2 amd64 [installed]
tpm-udev/jammy,now 0.6 all [installed]
ubuntu-minimal/jammy-updates,now 1.481.5 amd64 [installed]
ubuntu-server/jammy-updates,now 1.481.5 amd64 [installed]
ubuntu-standard/jammy-updates,now 1.481.5 amd64 [installed]
ufw/jammy-updates,now 0.36.1-4ubuntu0.1 all [installed]
unzip/jammy-updates,now 6.0-26ubuntu3.2 amd64 [installed]
usb-modeswitch-data/jammy,now 20191128-4 all [installed]
usb-modeswitch/jammy,now 2.6.1-3ubuntu2 amd64 [installed]
wget/jammy-updates,jammy-security,now 1.21.2-2ubuntu1.1 amd64 [installed]
xmlstarlet/jammy,now 1.6.1-2.1 amd64 [installed]
(END)

  UNIT                        LOAD   ACTIVE SUB     DESCRIPTION
  containerd.service          loaded active running containerd container runtime
  cron.service                loaded active running Regular background program processing daemon
  dbus.service                loaded active running D-Bus System Message Bus
  docker.service              loaded active running Docker Application Container Engine
  getty@tty1.service          loaded active running Getty on tty1
  multipathd.service          loaded active running Device-Mapper Multipath Device Controller
  networkd-dispatcher.service loaded active running Dispatcher daemon for systemd-networkd
  nginx.service               loaded active running A high performance web server and a reverse proxy server
  packagekit.service          loaded active running PackageKit Daemon
  polkit.service              loaded active running Authorization Manager
  qemu-guest-agent.service    loaded active running QEMU Guest Agent
  rsyslog.service             loaded active running System Logging Service
  serial-getty@ttyS0.service  loaded active running Serial Getty on ttyS0
  ssh.service                 loaded active running OpenBSD Secure Shell server
  systemd-journald.service    loaded active running Journal Service
  systemd-logind.service      loaded active running User Login Management
  systemd-networkd.service    loaded active running Network Configuration
  systemd-resolved.service    loaded active running Network Name Resolution
  systemd-timesyncd.service   loaded active running Network Time Synchronization
  systemd-udevd.service       loaded active running Rule-based Manager for Device Events and Files
  unattended-upgrades.service loaded active running Unattended Upgrades Shutdown
  user@0.service              loaded active running User Manager for UID 0

LOAD   = Reflects whether the unit definition was properly loaded.
ACTIVE = The high-level unit activation state, i.e. generalization of SUB.
SUB    = The low-level unit activation state, values depend on unit type.
22 loaded units listed.
UNIT FILE                              STATE   VENDOR PRESET
apparmor.service                       enabled enabled
blk-availability.service               enabled enabled
cloud-config.service                   enabled enabled
cloud-final.service                    enabled enabled
cloud-init-local.service               enabled enabled
cloud-init.service                     enabled enabled
console-setup.service                  enabled enabled
containerd.service                     enabled enabled
cron.service                           enabled enabled
dmesg.service                          enabled enabled
docker.service                         enabled enabled
e2scrub_reap.service                   enabled enabled
finalrd.service                        enabled enabled
getty@.service                         enabled enabled
grub-common.service                    enabled enabled
grub-initrd-fallback.service           enabled enabled
irqbalance.service                     enabled enabled
keyboard-setup.service                 enabled enabled
lvm2-monitor.service                   enabled enabled
lxd-agent.service                      enabled enabled
multipathd.service                     enabled enabled
networkd-dispatcher.service            enabled enabled
nginx.service                          enabled enabled
open-iscsi.service                     enabled enabled
open-vm-tools.service                  enabled enabled
pollinate.service                      enabled enabled
rsyslog.service                        enabled enabled
secureboot-db.service                  enabled enabled
setvtrgb.service                       enabled enabled
snapd.apparmor.service                 enabled enabled
snapd.autoimport.service               enabled enabled
snapd.core-fixup.service               enabled enabled
snapd.recovery-chooser-trigger.service enabled enabled
snapd.seeded.service                   enabled enabled
snapd.service                          enabled enabled
snapd.system-shutdown.service          enabled enabled
ssh.service                            enabled enabled
systemd-networkd-wait-online.service   enabled disabled
systemd-networkd.service               enabled enabled
systemd-pstore.service                 enabled enabled
systemd-resolved.service               enabled enabled
systemd-timesyncd.service              enabled enabled
ua-reboot-cmds.service                 enabled enabled
ubuntu-advantage.service               enabled enabled
ufw.service                            enabled enabled
unattended-upgrades.service            enabled enabled
vgauth.service                         enabled enabled

47 unit files listed.
NEXT                        LEFT          LAST                        PASSED       UNIT                           ACTIVATES
Mon 2026-03-16 13:33:28 UTC 1h 53min left Sun 2026-03-15 13:33:28 UTC 22h ago      update-notifier-download.timer update-notifier-download.service
Mon 2026-03-16 13:44:50 UTC 2h 4min left  Sun 2026-03-15 13:44:50 UTC 21h ago      systemd-tmpfiles-clean.timer   systemd-tmpfiles-clean.service
Mon 2026-03-16 13:52:29 UTC 2h 12min left Sun 2026-03-15 19:14:50 UTC 16h ago      apt-daily.timer                apt-daily.service
Mon 2026-03-16 14:53:41 UTC 3h 13min left Mon 2026-03-16 07:25:47 UTC 4h 14min ago motd-news.timer                motd-news.service
Mon 2026-03-16 21:56:40 UTC 10h left      Mon 2026-03-16 10:29:47 UTC 1h 10min ago certbot.timer                  certbot.service
Tue 2026-03-17 00:00:00 UTC 12h left      Mon 2026-03-16 00:00:05 UTC 11h ago      dpkg-db-backup.timer           dpkg-db-backup.service
Tue 2026-03-17 00:00:00 UTC 12h left      Mon 2026-03-16 00:00:05 UTC 11h ago      logrotate.timer                logrotate.service
Tue 2026-03-17 03:03:10 UTC 15h left      Mon 2026-03-16 05:51:05 UTC 5h 49min ago man-db.timer                   man-db.service
Tue 2026-03-17 03:18:55 UTC 15h left      Mon 2026-03-16 03:15:47 UTC 8h ago       pwl-market-backup.timer        pwl-market-backup.service
Tue 2026-03-17 06:26:11 UTC 18h left      Mon 2026-03-16 06:13:47 UTC 5h 26min ago apt-daily-upgrade.timer        apt-daily-upgrade.service
Tue 2026-03-17 12:59:54 UTC 1 day 1h left Tue 2026-03-10 08:48:05 UTC 6 days ago   update-notifier-motd.timer     update-notifier-motd.service
Sun 2026-03-22 03:10:11 UTC 5 days left   Sun 2026-03-15 03:10:47 UTC 1 day 8h ago e2scrub_all.timer              e2scrub_all.service
Mon 2026-03-23 00:35:51 UTC 6 days left   Mon 2026-03-16 01:10:47 UTC 10h ago      fstrim.timer                   fstrim.service
n/a                         n/a           n/a                         n/a          apport-autoreport.timer        apport-autoreport.service
n/a                         n/a           n/a                         n/a          snapd.snap-repair.timer        snapd.snap-repair.service
n/a                         n/a           n/a                         n/a          ua-timer.timer                 ua-timer.service

16 timers listed.
active
nginx: ACTIVE
inactive
postgresql: inactive/absent
inactive
redis: inactive/absent
inactive
traefik: inactive/absent
active
docker: ACTIVE
root@srv1178155:~#

Client: Docker Engine - Community
 Version:    29.2.1
 Context:    default
 Debug Mode: false
 Plugins:
  buildx: Docker Buildx (Docker Inc.)
    Version:  v0.31.1
    Path:     /usr/libexec/docker/cli-plugins/docker-buildx
  compose: Docker Compose (Docker Inc.)
    Version:  v5.0.2
    Path:     /usr/libexec/docker/cli-plugins/docker-compose

Server:
 Containers: 8
  Running: 8
  Paused: 0
  Stopped: 0
 Images: 9
 Server Version: 29.2.1
 Storage Driver: overlayfs
  driver-type: io.containerd.snapshotter.v1
 Logging Driver: json-file
 Cgroup Driver: systemd
 Cgroup Version: 2
 Plugins:
  Volume: local
  Network: bridge host ipvlan macvlan null overlay
  Log: awslogs fluentd gcplogs gelf journald json-file local splunk syslog
 CDI spec directories:
  /etc/cdi
  /var/run/cdi
 Swarm: inactive
 Runtimes: io.containerd.runc.v2 runc
 Default Runtime: runc
 Init Binary: docker-init
 containerd version: dea7da592f5d1d2b7755e3a161be07f43fad8f75
 runc version: v1.3.4-0-gd6d73eb8
 init version: de40ad0
 Security Options:
  apparmor
  seccomp
   Profile: builtin
  cgroupns
 Kernel Version: 5.15.0-164-generic
 Operating System: Ubuntu 22.04.5 LTS
 OSType: linux
 Architecture: x86_64
 CPUs: 1
 Total Memory: 3.819GiB
 Name: srv1178155
 ID: 7f89ff2a-8974-430b-930b-3584bb09b429
 Docker Root Dir: /var/lib/docker
 Debug Mode: false
 Experimental: false
 Insecure Registries:
  ::1/128
  127.0.0.0/8
 Live Restore Enabled: false
 Firewall Backend: iptables

NAMES             IMAGE                        STATUS                 PORTS
pwl-market-app    pwl-market-pwl-market-app    Up 10 days (healthy)   127.0.0.1:3101->3000/tcp
pwl-market-db     postgres:16-alpine           Up 13 days (healthy)   5432/tcp
pwl-more-app      pwl-more-pwl-more-app        Up 3 weeks (healthy)   127.0.0.1:3100->3000/tcp
pwl-more-db       postgres:16-alpine           Up 3 weeks (healthy)   5432/tcp
radio-rtmp-auth   radijas-v2-rtmp-auth         Up 3 weeks             8088/tcp
radio-autodj      radijas-v2-autodj            Up 3 weeks
radio-web         nginx:alpine                 Up 3 weeks             0.0.0.0:8080->80/tcp, [::]:8080->80/tcp
radio-rtmp        tiangolo/nginx-rtmp:latest   Up 3 weeks (healthy)   0.0.0.0:1935->1935/tcp, [::]:1935->1935/tcp
NAMES             IMAGE                        STATUS
pwl-market-app    pwl-market-pwl-market-app    Up 10 days (healthy)
pwl-market-db     postgres:16-alpine           Up 13 days (healthy)
pwl-more-app      pwl-more-pwl-more-app        Up 3 weeks (healthy)
pwl-more-db       postgres:16-alpine           Up 3 weeks (healthy)
radio-rtmp-auth   radijas-v2-rtmp-auth         Up 3 weeks
radio-autodj      radijas-v2-autodj            Up 3 weeks
radio-web         nginx:alpine                 Up 3 weeks
radio-rtmp        tiangolo/nginx-rtmp:latest   Up 3 weeks (healthy)
NETWORK ID     NAME                 DRIVER    SCOPE
390074d6ab3a   bridge               bridge    local
f287e310837b   host                 host      local
0bd9cefa822a   none                 null      local
6439d3adc099   pwl-market-net       bridge    local
5c69ded7dae0   pwl-more-net         bridge    local
7b750bdf7e99   radijas-v2_default   bridge    local
DRIVER    VOLUME NAME
local     radijas-v2_hls-data
local     radijas-v2_status-data
REPOSITORY                      TAG         SIZE
pwl-market-pwl-market-app       latest      274MB
pwl-market-pwl-market-migrate   latest      704MB
pwl-more-pwl-more-app           latest      248MB
radijas-v2-rtmp-auth            latest      177MB
radijas-v2-autodj               latest      814MB
tiangolo/nginx-rtmp             latest      1.25GB
postgres                        16-alpine   395MB
nginx                           alpine      93.4MB
node                            20-alpine   192MB
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          9         7         9.011GB   8.403GB (93%)
Containers      8         8         1.707GB   0B (0%)
Local Volumes   0         0         0B        0B
Build Cache     47        0         4.09GB    1.211GB
root@srv1178155:~#
root@srv1178155:~# # Listening ports (all protocols)
ss -tlnp
# or:
netstat -tlnp 2>/dev/null || echo "netstat not available"

# Firewall rules
ufw status verbose 2>/dev/null || iptables -L -n --line-numbers 2>/dev/null

# Public IP
curl -s https://ifconfig.me; echo
State                      Recv-Q                     Send-Q                                         Local Address:Port                                          Peer Address:Port                     Process
LISTEN                     0                          4096                                           127.0.0.53%lo:53                                                 0.0.0.0:*                         users:(("systemd-resolve",pid=577,fd=14))
LISTEN                     0                          511                                                127.0.0.1:8088                                               0.0.0.0:*                         users:(("nginx",pid=3489053,fd=6),("nginx",pid=335733,fd=6))
LISTEN                     0                          511                                                127.0.0.1:8089                                               0.0.0.0:*                         users:(("nginx",pid=3489053,fd=7),("nginx",pid=335733,fd=7))
LISTEN                     0                          511                                                  0.0.0.0:443                                                0.0.0.0:*                         users:(("nginx",pid=3489053,fd=10),("nginx",pid=335733,fd=10))
LISTEN                     0                          511                                                  0.0.0.0:80                                                 0.0.0.0:*                         users:(("nginx",pid=3489053,fd=8),("nginx",pid=335733,fd=8))
LISTEN                     0                          128                                                  0.0.0.0:22                                                 0.0.0.0:*                         users:(("sshd",pid=1399557,fd=3))
LISTEN                     0                          4096                                               127.0.0.1:3100                                               0.0.0.0:*                         users:(("docker-proxy",pid=2613007,fd=8))
LISTEN                     0                          4096                                               127.0.0.1:3101                                               0.0.0.0:*                         users:(("docker-proxy",pid=516808,fd=8))
LISTEN                     0                          4096                                                 0.0.0.0:8080                                               0.0.0.0:*                         users:(("docker-proxy",pid=2372150,fd=8))
LISTEN                     0                          4096                                                 0.0.0.0:1935                                               0.0.0.0:*                         users:(("docker-proxy",pid=2372190,fd=8))
LISTEN                     0                          511                                                     [::]:80                                                    [::]:*                         users:(("nginx",pid=3489053,fd=9),("nginx",pid=335733,fd=9))
LISTEN                     0                          128                                                     [::]:22                                                    [::]:*                         users:(("sshd",pid=1399557,fd=4))
LISTEN                     0                          4096                                                    [::]:8080                                                  [::]:*                         users:(("docker-proxy",pid=2372155,fd=8))
LISTEN                     0                          4096                                                    [::]:1935                                                  [::]:*                         users:(("docker-proxy",pid=2372195,fd=8))
Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
tcp        0      0 127.0.0.53:53           0.0.0.0:*               LISTEN      577/systemd-resolve
tcp        0      0 127.0.0.1:8088          0.0.0.0:*               LISTEN      335733/nginx: worke
tcp        0      0 127.0.0.1:8089          0.0.0.0:*               LISTEN      335733/nginx: worke
tcp        0      0 0.0.0.0:443             0.0.0.0:*               LISTEN      335733/nginx: worke
tcp        0      0 0.0.0.0:80              0.0.0.0:*               LISTEN      335733/nginx: worke
tcp        0      0 0.0.0.0:22              0.0.0.0:*               LISTEN      1399557/sshd: /usr/
tcp        0      0 127.0.0.1:3100          0.0.0.0:*               LISTEN      2613007/docker-prox
tcp        0      0 127.0.0.1:3101          0.0.0.0:*               LISTEN      516808/docker-proxy
tcp        0      0 0.0.0.0:8080            0.0.0.0:*               LISTEN      2372150/docker-prox
tcp        0      0 0.0.0.0:1935            0.0.0.0:*               LISTEN      2372190/docker-prox
tcp6       0      0 :::80                   :::*                    LISTEN      335733/nginx: worke
tcp6       0      0 :::22                   :::*                    LISTEN      1399557/sshd: /usr/
tcp6       0      0 :::8080                 :::*                    LISTEN      2372155/docker-prox
tcp6       0      0 :::1935                 :::*                    LISTEN      2372195/docker-prox
Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), deny (routed)
New profiles: skip

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW IN    Anywhere
80/tcp                     ALLOW IN    Anywhere
443/tcp                    ALLOW IN    Anywhere
1935/tcp                   ALLOW IN    Anywhere
22/tcp (v6)                ALLOW IN    Anywhere (v6)
80/tcp (v6)                ALLOW IN    Anywhere (v6)
443/tcp (v6)               ALLOW IN    Anywhere (v6)
1935/tcp (v6)              ALLOW IN    Anywhere (v6)

2a02:4780:41:8172::1
root@srv1178155:~#
root@srv1178155:~# # nginx — check if installed and what it's serving
nginx -t 2>/dev/null
ls /etc/nginx/sites-enabled/ 2>/dev/null
cat /etc/nginx/sites-enabled/default 2>/dev/null || echo "no default nginx site"
ls /etc/nginx/conf.d/ 2>/dev/null

# Traefik — host-level install
which traefik && traefik version 2>/dev/null || echo "traefik: not on PATH"
ls /etc/traefik/ 2>/dev/null || echo "/etc/traefik: absent"

# Caddy
which caddy && caddy version 2>/dev/null || echo "caddy: absent"

# Certificates (Let's Encrypt)
ls /etc/letsencrypt/live/ 2>/dev/null || echo "certbot: no live certs"
ls /root/.acme.sh/ 2>/dev/null || echo "acme.sh: absent"
00-default.conf  market.peoplewelike.club  more.peoplewelike.club.conf  radio.peoplewelike.club.conf
no default nginx site
rtmp_auth.conf  rtmp_stat.conf
traefik: not on PATH
/etc/traefik: absent
caddy: absent
README  market.peoplewelike.club  more.peoplewelike.club  radio.peoplewelike.club
acme.sh: absent
root@srv1178155:~#
/opt/niteos: absent
containerd  pwl-market  pwl-more  radijas-v2
hls  hls_v2  html  radio  radio.peoplewelike.club
ubuntu
/opt/pwl-more/.git
/opt/radijas-v2/.git
/opt/pwl-market/.git
/root/more-build/.git
/root/member/.git
/root/radijas/.git
root@srv1178155:~#

root@srv1178155:~# # NiteOS env files (NEVER print contents — just confirm presence)
for f in \
  /opt/niteos/infra/cloud.env \
  /opt/niteos/infra/secrets/jwt_private_key.pem \
  /opt/niteos/infra/secrets/jwt_public_key.pem \
  /opt/niteos/infra/traefik/acme.json \
  /opt/niteos/backup.env; do
  if [ -f "$f" ]; then
    echo "EXISTS  $(stat -c '%a %n' $f)"
  else
    echo "ABSENT  $f"
  fi
done
ABSENT  /opt/niteos/infra/cloud.env
ABSENT  /opt/niteos/infra/secrets/jwt_private_key.pem
ABSENT  /opt/niteos/infra/secrets/jwt_public_key.pem
ABSENT  /opt/niteos/infra/traefik/acme.json
ABSENT  /opt/niteos/backup.env
root@srv1178155:~#

root@srv1178155:~# # Host-level Postgres (if any)
if command -v psql &>/dev/null; then
  sudo -u postgres psql -c "\l" 2>/dev/null || echo "postgres: no local cluster"
fi

# Dockerised Postgres (if Docker is up)
COMPOSE="docker compose -f /opt/niteos/infra/docker-compose.cloud.yml --env-file /opt/niteos/infra/cloud.env"
$COMPOSE exec postgres psql -U niteos -d niteos -c "\dn" 2>/dev/null || echo "docker postgres: not reachable"

# Postgres data volume
docker volume inspect niteos_pgdata 2>/dev/null || echo "pgdata volume: absent"
docker postgres: not reachable
[]
pgdata volume: absent
root@srv1178155:~#
root@srv1178155:~# # All Docker volumes and their mount points
docker volume ls -q | xargs -I{} docker volume inspect {} --format '{{.Name}} → {{.Mountpoint}} ({{.Driver}})' 2>/dev/null

# Disk usage per volume (may require root)
du -sh /var/lib/docker/volumes/*/  2>/dev/null | sort -rh | head -20
radijas-v2_hls-data → /var/lib/docker/volumes/radijas-v2_hls-data/_data (local)
radijas-v2_status-data → /var/lib/docker/volumes/radijas-v2_status-data/_data (local)
297M    /var/lib/docker/volumes/radijas-v2_hls-data/
32K     /var/lib/docker/volumes/radijas-v2_status-data/
root@srv1178155:~#
/opt/niteos/backups: absent
niteos-backup.timer: absent
niteos-backup.service: absent
backup.env: absent
rclone: not installed or not configured
root@srv1178155:~#
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        49G   14G   35G  29% /
3.3G    /var/lib/docker
531M    /opt/pwl-market
204M    /var/log
2.9M    /opt/pwl-more
1.8M    /opt/radijas-v2
32K     /home/ubuntu
12K     /opt/containerd
Images space usage:

REPOSITORY                      TAG         IMAGE ID       CREATED       SIZE      SHARED SIZE   UNIQUE SIZE   CONTAINERS
pwl-market-pwl-market-app       latest      ab79f4bc8ff3   10 days ago   274MB     143.7MB       130.2MB       1
pwl-market-pwl-market-migrate   latest      414d245f62f7   12 days ago   704MB     143.7MB       560.1MB       0
pwl-more-pwl-more-app           latest      7deada7e98b4   3 weeks ago   248MB     143.7MB       104MB         1
radijas-v2-rtmp-auth            latest      3de825c01c35   3 weeks ago   177MB     87.43MB       89.51MB       1
radijas-v2-autodj               latest      11b0c95d527d   3 weeks ago   814MB     87.43MB       726.9MB       1
tiangolo/nginx-rtmp             latest      627c00f73343   4 weeks ago   1.25GB    0B            1.246GB       1
postgres                        16-alpine   97ff59a4e30e   4 weeks ago   395MB     9.105MB       386.3MB       2
nginx                           alpine      1d13701a5f9f   5 weeks ago   93.4MB    9.105MB       84.31MB       1
node                            20-alpine   09e2b3d97260   6 weeks ago   192MB     143.7MB       48.32MB       0

Containers space usage:

CONTAINER ID   IMAGE                        COMMAND                  LOCAL VOLUMES   SIZE      CREATED       STATUS                 NAMES
f836bf5525fb   pwl-market-pwl-market-app    "docker-entrypoint.s…"   0               160kB     10 days ago   Up 10 days (healthy)   pwl-market-app
4f231893f893   postgres:16-alpine           "docker-entrypoint.s…"   0               24.6kB    13 days ago   Up 13 days (healthy)   pwl-market-db
4ba77071e15e   pwl-more-pwl-more-app        "docker-entrypoint.s…"   0               108MB     3 weeks ago   Up 3 weeks (healthy)   pwl-more-app
da10f18bbaae   postgres:16-alpine           "docker-entrypoint.s…"   0               106MB     3 weeks ago   Up 3 weeks (healthy)   pwl-more-db
a5db5cb467ae   radijas-v2-rtmp-auth         "python -u auth.py"      0               153MB     3 weeks ago   Up 3 weeks             radio-rtmp-auth
6b1986bc91e4   radijas-v2-autodj            "/app/overlay.sh"        1               429MB     3 weeks ago   Up 3 weeks             radio-autodj
46bd6275f081   nginx:alpine                 "/docker-entrypoint.…"   2               72MB      3 weeks ago   Up 3 weeks             radio-web
f0d8bc093c98   tiangolo/nginx-rtmp:latest   "nginx -g 'daemon of…"   1               839MB     3 weeks ago   Up 3 weeks (healthy)   radio-rtmp

Local Volumes space usage:

VOLUME NAME   LINKS     SIZE

Build cache usage: 4.09GB

CACHE ID       CACHE TYPE     SIZE      CREATED       LAST USED     USAGE     SHARED
c5mjr6tjm0oc   regular        3.86MB    12 days ago   12 days ago   1         true
d47pjnrop8ji   regular        42.8MB    12 days ago   12 days ago   1         true
l5ub6w4il5lq   regular        1.26MB    12 days ago   12 days ago   1         true
s1l8lmuxnhtn   regular        808MB     12 days ago   12 days ago   1         true
uo5g9z5vlafq   regular        46kB      12 days ago   12 days ago   1         false
azvg1uxnzf8n   regular        12.3kB    12 days ago   12 days ago   1         true
ll6hx9zkzifs   regular        16.5kB    12 days ago   12 days ago   1         false
nxn4gepew0iz   regular        8.19kB    12 days ago   12 days ago   1         true
x2r09fv37z1j   regular        55.9MB    12 days ago   12 days ago   1         false
r98wdxxmfrmx   regular        1.24MB    12 days ago   12 days ago   1         false
tntdi3iucz2i   regular        114MB     12 days ago   12 days ago   1         true
qk2g6cf8w0g9   regular        20.9kB    12 days ago   12 days ago   1         false
d276cvk58c02   regular        313MB     12 days ago   12 days ago   1         true
jged7ci3f6cr   regular        815kB     12 days ago   12 days ago   1         true
nw70t1jkudz4   regular        1.25MB    10 days ago   10 days ago   1         false
ifukzjny6cxr   regular        114MB     10 days ago   10 days ago   1         false
xf49n4okbb7e   regular        55.9MB    10 days ago   10 days ago   1         false
yqri5t10odnx   regular        16.5kB    10 days ago   10 days ago   1         false
op808pzmehpe   regular        762kB     10 days ago   10 days ago   1         false
epwzybqpl6ew   regular        766kB     10 days ago   10 days ago   2         false
jav2j8wavjnr   regular        114MB     10 days ago   10 days ago   1         false
zjrs5egkh84i   regular        1.25MB    10 days ago   10 days ago   1         false
qm48ghal4kh6   regular        511MB     12 days ago   10 days ago   4         true
mxjsf75kykrg   regular        762kB     10 days ago   10 days ago   1         false
u2tva37htqhd   regular        55.9MB    10 days ago   10 days ago   1         false
mtm5o4b9ynhh   regular        16.5kB    10 days ago   10 days ago   1         false
kexhqa7u0cuh   regular        106kB     10 days ago   10 days ago   1         true
09frhiapng6m   regular        8.19kB    10 days ago   10 days ago   1         true
3ox9o0l2m1l5   regular        1.14MB    10 days ago   10 days ago   1         false
qaycg6gmw4vf   regular        313MB     10 days ago   10 days ago   1         true
vprow9fkflyd   regular        45.1kB    10 days ago   10 days ago   1         false
whh5pn74tnpf   regular        771MB     10 days ago   10 days ago   1         true
bah73i629119   regular        445B      12 days ago   10 days ago   2         true
fp3qtdqssrxr   regular        8.29kB    12 days ago   10 days ago   2         true
sr6k11y8rxgl   regular        20.5kB    10 days ago   10 days ago   1         false
drc0ze3khd9c   regular        1.14MB    10 days ago   10 days ago   1         false
s5o6g3k1rfdn   regular        1.14MB    10 days ago   10 days ago   1         false
jejs9upnvx29   source.local   1.13MB    12 days ago   10 days ago   9         false
cofxn4eitufg   regular        1.14MB    10 days ago   10 days ago   1         false
yy4ni678hmml   regular        1.31MB    10 days ago   10 days ago   1         false
j0kwuulab85a   regular        12.4kB    12 days ago   10 days ago   4         false
jgffxdmwkg4s   regular        571MB     10 days ago   10 days ago   4         false
i6jd70ghx7vf   regular        80.9MB    10 days ago   10 days ago   1         false
lv4tbovxypq6   regular        16.5kB    10 days ago   10 days ago   1         false
6nn8zr9tf7nm   regular        151MB     10 days ago   10 days ago   1         false
swe19udydzyz   source.local   8.19kB    12 days ago   10 days ago   9         false
f72te08rlq2b   source.local   8.19kB    12 days ago   10 days ago   9         false
root@srv1178155:~#
root@srv1178155:~# # Identify what service-1 is
ls /opt/service-1/ 2>/dev/null || echo "no /opt/service-1"

# Any nginx vhosts pointing to service-1
grep -r "service-1\|service1" /etc/nginx/ 2>/dev/null || echo "no nginx config referencing service-1"

# Process list — find anything running on 80/443
ss -tlnp | grep -E ':80|:443'
lsof -i :80 -i :443 2>/dev/null | grep LISTEN
no /opt/service-1
no nginx config referencing service-1
LISTEN 0      511        127.0.0.1:8088      0.0.0.0:*    users:(("nginx",pid=3489053,fd=6),("nginx",pid=335733,fd=6))
LISTEN 0      511        127.0.0.1:8089      0.0.0.0:*    users:(("nginx",pid=3489053,fd=7),("nginx",pid=335733,fd=7))
LISTEN 0      511          0.0.0.0:443       0.0.0.0:*    users:(("nginx",pid=3489053,fd=10),("nginx",pid=335733,fd=10))
LISTEN 0      511          0.0.0.0:80        0.0.0.0:*    users:(("nginx",pid=3489053,fd=8),("nginx",pid=335733,fd=8))
LISTEN 0      4096         0.0.0.0:8080      0.0.0.0:*    users:(("docker-proxy",pid=2372150,fd=8))
LISTEN 0      511             [::]:80           [::]:*    users:(("nginx",pid=3489053,fd=9),("nginx",pid=335733,fd=9))
LISTEN 0      4096            [::]:8080         [::]:*    users:(("docker-proxy",pid=2372155,fd=8))
nginx    335733 www-data    8u  IPv4 43575833      0t0  TCP *:http (LISTEN)
nginx    335733 www-data    9u  IPv6 43575834      0t0  TCP *:http (LISTEN)
nginx    335733 www-data   10u  IPv4 43575835      0t0  TCP *:https (LISTEN)
nginx   3489053     root    8u  IPv4 43575833      0t0  TCP *:http (LISTEN)
nginx   3489053     root    9u  IPv6 43575834      0t0  TCP *:http (LISTEN)
nginx   3489053     root   10u  IPv4 43575835      0t0  TCP *:https (LISTEN)
root@srv1178155:~#

Item	Value
OS version	Ubuntu 22.04.5 LTS
Kernel	5.15.0-164-generic
Docker installed	yes
Docker version	29.2.1
nginx running	yes
nginx ports	80, 443, 8088, 8089
Traefik (host)	no
/opt/niteos exists	no
NiteOS repo commit	N/A
cloud.env present	no
JWT keys present	no
acme.json present	no
Docker pgdata volume	no
Postgres schemas present	no
Local backups present	no
Disk free on /	35G
service-1 running	no
service-1 port	N/A


