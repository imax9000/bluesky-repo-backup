# Backup for your Bluesky account data

Runs periodically and publishes your ATproto repo to GitHub Pages.

## How to use this?

1. Fork this repo
2. Clone your fork to a local machine (for one-time setup)
3. Copy `.env.example` to `.env` and update the content with your info. `TOKEN` will be filled in later if needed.
4. Run `make` to generate a new rotation key.
5. Add `PRIVATE_KEY` under Settings -> Secrets and variables -> Actions in the Secrets tab.
6. Add `PUBLIC_KEY` and `DID` (with your DID as a value) in the Variables tab.
7. Run `make add-key` to add your newly generated key to the list of rotation keys in PLC.
    * It may tell you to check your email. If it does - copy the token from the email from your PDS into `TOKEN` variable in `.env` and re-run the command.

To err on the side of caution, you can now delete `key.priv` file.

You can trigger the workflow manually too, from the Actions tab. Your repo will be published as `${REPO_PAGES_URL}/xrpc/com.atproto.sync.getRepo`, mimicking XRPC method.

## Caution: missing features

TODO:

* Verifying repo signature with the public key from DID document
* Fetching blobs: might easily overrun Pages size limits, so probably worth to store them elsewhere
* Re-adding `PUBLIC_KEY` to PLC if it gets removed for whatever reason or bumped from the top of the list
* Reverting PLC operations when `PUBLIC_KEY` gets removed and your current PDS refuses to cooperate
* A script for migrating your account to a new PDS from the last backup
