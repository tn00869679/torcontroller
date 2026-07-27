# iptable

Using iptables-legacy

## Command

Commands for redirection

```bash
# Redirect all HTTP/HTTPS traffic to Privoxy's 8118 port (IPv4)
iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8118
iptables -t nat -A OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8118

# Redirect all HTTP/HTTPS traffic to Privoxy's 8118 port (IPv6)
ip6tables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8118
ip6tables -t nat -A OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8118
```

Commands to clear rules

```bash
# Remove the rules for IPv4
iptables -t nat -D OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8118
iptables -t nat -D OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8118

# Remove the rules for IPv6
ip6tables -t nat -D OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8118
ip6tables -t nat -D OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8118
```
