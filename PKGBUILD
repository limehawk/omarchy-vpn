# Maintainer: Limehawk <limehawk@users.noreply.github.com>
pkgname=omarchy-vpn
pkgver=0.1.0
pkgrel=1
pkgdesc="WireGuard VPN manager TUI for Omarchy"
arch=('x86_64')
url="https://github.com/limehawk/omarchy-vpn"
license=('MIT')
depends=('wireguard-tools' 'systemd-resolvconf')
makedepends=('go')
install=omarchy-vpn.install

build() {
    cd "$startdir"
    go build -o "$srcdir/omarchy-vpn" .
}

package() {
    cd "$srcdir"

    # Binary
    install -Dm755 omarchy-vpn "$pkgdir/usr/bin/omarchy-vpn"

    # Sudoers for passwordless WireGuard management
    install -Dm440 /dev/stdin "$pkgdir/etc/sudoers.d/omarchy-vpn" << 'EOF'
%wheel ALL=(ALL) NOPASSWD: /usr/bin/wg-quick up *
%wheel ALL=(ALL) NOPASSWD: /usr/bin/wg-quick down *
%wheel ALL=(ALL) NOPASSWD: /usr/bin/wg show *
%wheel ALL=(ALL) NOPASSWD: /usr/bin/ls /etc/wireguard
%wheel ALL=(ALL) NOPASSWD: /usr/bin/cat /etc/wireguard/*.conf
%wheel ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/wireguard/*.conf
%wheel ALL=(ALL) NOPASSWD: /usr/bin/chmod 600 /etc/wireguard/*.conf
%wheel ALL=(ALL) NOPASSWD: /usr/bin/mv /etc/wireguard/*.conf /etc/wireguard/*.conf
%wheel ALL=(ALL) NOPASSWD: /usr/bin/rm /etc/wireguard/*.conf
EOF
}
