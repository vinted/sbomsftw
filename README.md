# SBOMS For The Win!

Tool for SBOM (Software Bill Of Materials) collection from filesystems & GitHub repositories.

## Building
Since SBOM collection from repositories depends on many external tools - it's **highly recommended** to collect SBOMs from inside Docker containers, where all the tools are already packaged.
Clone the repo & build the docker image from the repository root with:
```bash
docker build --tag sbomsftw -f build/Dockerfile .
```

## Configuration (Optional)
To Collect SBOMs from private GitHub repositories, a valid set of credentials must be provided. This is done via environment variables. E.g:
```bash
export SAC_GITHUB_USERNAME=your-username
export SAC_GITHUB_TOKEN=personal-access-token-with-read-scope
```
Also, to upload SBOMs to Dependency Track, a valid API Token and Dependency Track API base URL must be provided. This is done via environment variables as well. E.g:
```bash
export SAC_DEPENDENCY_TRACK_TOKEN=dependency-track-access-token-with-write-scope
export SAC_DEPENDENCY_TRACK_URL=https://api-dependency-track.evilcorp.com/
```
**Note:**\
When collecting SBOMs inside a Docker container - format your env variables like this:
```bash
SAC_DEPENDENCY_TRACK_TOKEN=dependency-track-access-token-with-write-scope
SAC_DEPENDENCY_TRACK_URL=https://api-dependency-track.evilcorp.com/
SAC_GITHUB_USERNAME=your-username
SAC_GITHUB_TOKEN=personal-access-token-with-read-scope
```
And then pass them to `docker run` with `--env-file` switch.

## Examples
Single repository mode (output is stored at `outputs/sboms.json`):
```bash
docker run -it --rm -v "${PWD}/outputs/":'/tmp/' sbomsftw:latest sa-collector repo https://github.com/cloudflare/quiche \
	--output /tmp/sboms.json --log-format fancy
```
Organization mode - collect SBOMs from every repository inside the organization & upload them to Dependency Track:
```bash
docker run --env-file .env -it --rm sbomsftw:latest sa-collector org https://api.github.com/orgs/vinted/repos \
        --output /dev/null --tags 'vinted' --upload-to-dependency-track --log-format fancy
```

Filesystem collection mode:
```
sa-collector fs / --exclude './usr/local/bin' --exclude './root' --exclude './etc'  --exclude './dev' --output sboms.json
```
**Note:**\
Filesystem scans exclude files relative to the specified directory. For example: scanning `/usr/foo` with `--exclude ./package.json` would exclude `/usr/foo/package.json` and `--exclude '**/package.json'` would exclude all `package.json` files under `/usr/foo`. For filesystem scans, it is required to begin path expressions with `./`, `*/`, or `**/`, all of which will be resolved relative to the specified scan directory. Keep in mind, your shell may attempt to expand wildcards, so put those parameters in single quotes, like: '**/*.json'.

------
```bash
Collects CycloneDX SBOMs from Github repositories

Usage:
  subcommand [command]

Examples:
sa-collector repo https://github.com/ReactiveX/RxJava                  collect SBOMs from RxJava repository & output them to stdout
sa-collector repo https://github.com/ffuf/ffuf --output sboms.json     collect SBOMs from ffuf repository & write results to sboms.json

sa-collector org https://api.github.com/orgs/evil-corp/repos           collect SBOMs from evil-corp organization & output them to stdout

sa-collector fs /usr/local/bin --upload-to-dependency-track            collect SBOMs recursively from /usr/local/bin directory & upload them to Dependency Track
sa-collector fs / --exclusions './root'                                collect SBOMs recursively from root directory while excluding /root directory

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  fs          Collect SBOMs from a filesystem path
  help        Help about any command
  org         Collect SBOMs from every repository inside the given organization
  repo        Collect SBOMs from a single repository

Flags:
  -h, --help                         help for subcommand
  -f, --log-format string            log format: simple/fancy/json (default "simple")
  -l, --log-level string             log level: debug/info/warn/error/fatal/panic (default "info")
  -o, --output string                where to output SBOM results: (defaults to stdout when unspecified)
  -t, --tags strings                 tags to use when SBOMs are uploaded to Dependency Track (optional)
  -u, --upload-to-dependency-track   whether to upload collected SBOMs to Dependency Track (default: false)

Use "subcommand [command] --help" for more information about a command.
```
