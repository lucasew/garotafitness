# ADR-0003: setup.exe is data

Status: accepted

`setup.exe` is read for Encoder names. Volume headers are the primary names.

Rejected execution of installer PE or DLL bytes. Rejected Wine. Rejected `innoextract-go` as the 5.5 parser; it accepts Inno 6.x only.
