# package devdns

This package provides an installer and an implementation of a DNS
server that can serve requests for a given list of tlds.

If you configure the server to work with `test` then any DNS lookup that
ends with `.test` will resolve to localhost.  Eg. `example.test`.

## Credits

The majority of the code for this package was extracted from
https://github.com/puma/puma-dev.
