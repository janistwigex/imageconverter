# Test fixtures

The tests in this package run against any RAW files placed in this directory.
RAW sample files are **not committed** to the repository (they are large and
their licensing varies), so the tests will **skip** automatically when no
fixture is present.

To run the full test suite locally, drop one or more RAW files here, for example:

```
tests/testdata/sample.nef
tests/testdata/sample.cr2
```

Good sources of freely licensed sample RAW files:

- https://raw.pixls.us — community RAW database (files are CC0)
- https://www.rawsamples.ch — RAW samples for many cameras
