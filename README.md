# ffxf-terraform-provider
The terraform provider for ffxf

## OpenTofu releases

Pushing a version tag such as `v1.2.3` runs GoReleaser and publishes the platform archives, a SHA-256 checksum file, and its GPG signature to GitHub Releases. OpenTofu Registry can then consume that release after the provider has been registered there.

Configure the repository secrets `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` with the release signing key. Publish the matching public key with the OpenTofu Registry provider listing so it can verify the signed checksums.
