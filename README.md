# exitd
A simpe process supervisor for containers.

## What it does
exitd does one thing:
- run two or more processes and exit with an error if any of them exits.

## But why?
Sometimes you need to run two or more processes in a container (say nginx and php-fpm) but you still want it
to behave in a container-friendly manner.

## Using
exitd is intended to be used with [tini][tini], [dumb-init][dumb-init] or similar. It's not a replacement
for a proper `init` since it does none of the signal handling or process reaping
that `init` needs to do.

Typical Docker usage with tini:

```docker
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/sbin/exitd", "/usr/bin/nginx", "/usr/bin/php-fpm"]
```

Note that it's currently not possible to supply any command-line arguments
to your programs. If you need to pass arguments you can create wrapper scripts.
Please note the importance of using `exec` in these scripts to keep signal handling working.
For example:

```docker
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/sbin/exitd", "/root/scripts/nginx", "/root/scripts/php-fpm"]
```

`/root/scripts/nginx`
```shell
#!/bin/sh
exec /usr/sbin/nginx -c /etc/my-nginx-config.conf
```

`/root/scripts/php-fpm`
```shell
#!/bin/sh
exec /usr/sbin/php-fpm -y /etc/my-php-fpm-config.conf
```

[tini]: https://github.com/krallin/tini
[dumb-init]: https://github.com/Yelp/dumb-init
