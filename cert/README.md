# package cert

This package contains functions for creating a local development root
certificate that can then be used with net/http tlsConfig servers to dynamically
generate SSL certificates in development.

The generated certificates are kept in-memory in a cache and are re-generated
on next execution.

On macOS the certificate is installed into the system keychain (a password
prompt is shown) so all generated certificates are trusted automatically.

## Usage example

You need to run the root certificate creation step once:

```
  err := cert.CreateCert()
```

Now you can create an HTTPS server that uses the root certificate to generate
valid certificates dynamically:

```
  cache, err := cert.NewCache()

  tlsConfig := &tls.Config{
    GetCertificate: cache.GetCertificate,
  }
```

See https://github.com/moomerman/zap/tree/master/cert/example_test.go for
a full example.

## Credits

The majority of the code for this package was extracted from
https://github.com/puma/puma-dev.
