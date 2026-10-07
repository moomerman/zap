# package dns

This package provides a DNS responder that answers requests for a given list
of tlds, and helpers for the macOS resolver files that send lookups to it.

If you configure the server to work with `test` then any DNS lookup that
ends with `.test` will resolve to localhost.  Eg. `example.test`.

## Credits

The majority of the code for this package was extracted from
https://github.com/puma/puma-dev.
