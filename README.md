# Challendar

Estimated difficulty: 6 hours

## Summary

This challenge centres around the CalDAV protocol, a superset of the DAV protocol which is itself a superset of the HTTP protocol, featuring additional HTTP verbs such as MKCALENDAR, REPORT, MOVE, COPY and so on. This challenge tests the user's web code review skills as well as their ability to parse the CalDAV RFCs. It's also an interesting twist on typical web challenges as this time you're working in CalDAV, not HTTP verbs.

The challenge consists of two services - a Radicale CalDAV server and an in-progress self-written Golang CalDAV server. The user will be provided the Golang CalDAV server code, but Radicale can be fingerprinted with the "Radicale works!" message at the default `/.web` endpoint (root redirects here).

Radicale CalDAV server is a popular Python CalDAV server. However, it unsafely deserializes sync tokens in its REPORT handler:

`radicale\storage\multifilesystem\sync.py`
```
token_folder = os.path.join(self._filesystem_path,
                                    ".Radicale.cache", "sync-token")
        token_path = os.path.join(token_folder, token_name)
        old_state = {}
        if old_token_name:
            # load the old token state
            old_token_path = os.path.join(token_folder, old_token_name)
            try:
                # Race: Another process might have deleted the file.
                with open(old_token_path, "rb") as f:
                    old_state = pickle.load(f)
            except (FileNotFoundError, pickle.UnpicklingError,
                    ValueError) as e:
                if isinstance(e, (pickle.UnpicklingError, ValueError)):
                    logger.warning(
                        "Failed to load stored sync token %r in %r: %s",
                        old_token_name, self.path, e, exc_info=True)
                    # Delete the damaged file
                    with contextlib.suppress(FileNotFoundError,
                                             PermissionError):
                        os.remove(old_token_path)
                raise ValueError("Token not found: %r" % old_token)
```

However, it uses several sanitizers to ensure that the `<CALENDAR ROOT>/.Radicale.cache/sync-token/<SYNC TOKEN NAME>` path is not writable by a user ordinarily.

Unfortunately, the self-written Golang CalDAV server shares the same calendar root folder as Radicale, oestinsibly because the victim is building a backward-compatible Golang replacement for Radicale. The replacement is built on top of the official `golang.org/x/net/webdav` package. Since CalDAV/WebDAV supports uploading files to the root, an attacker can exploit this by uploading the pickle payload to the appropriate sync token path, then triggering the payload with an appropriate REPORT request to Radicale.

To make this more difficult, I blocked uploading directly to the sync token path by limiting the number of path parts in any request to the Golang server:

```
func (h *Handler) checkIsAuthorized(req *http.Request) (int, error) {
	// should already be authorized
	username, _, _ := req.BasicAuth()
	urlParts := strings.Split(req.URL.Path, "/")
	// users can only access their own resources
	if username != urlParts[1] || len(urlParts) > 4 {
		return http.StatusUnauthorized, errUnauthorized
	}
	return http.StatusOK, nil
}
```

I also disabled several CalDAV verbs that might make enumeration easier:

```
            // To update to CalDAV RFC
			case "PROPFIND", "PROPPATCH", "MKCALENDAR", "REPORT":
				status = http.StatusNotImplemented
				err = errNotImplemented
			}
```

As such, the only path for an attacker is by using a `PUT` request to upload the pickle payload, then using the `Destination` header in a `MOVE` or `COPY` request (which is not inspected by `checkIsAuthorized`) to move the payload to the sync token path.

There are a few additional built-in checks by Radicale itself:

```
def check_token_name(token_name: str) -> bool:
    if len(token_name) != 64:
        return False
    for c in token_name:
        if c not in "0123456789abcdef":
            return False
    return True
```

The participant must properly read through the Radicale code to notice these. Finally, they must craft the `REPORT` request properly (Radicale uses a strange `http://radicale.org/ns/sync/TOKEN_NAME` format for the `sync-token` node) to trigger it.

In short, the participant must be able to discover two separate almost-bugs in Radicale and the Golang server, then chain them into an actual vulnerability.

## Setup

I have created the user `jrarj`/`H3110fr13nD` for `http://calendar:3000` in the Thunderbird backup file. To modify this, change the `logins.json` file such that `http://calendar:3000` points to the actual Radicale server and `http://devcalendar:4000` points to the Golang CALDAV server.

This is a simple forensics challenge; `python3 firepwd.py -d <PATH TO EXTRACTED BACKUP FOLDER>` is sufficient to return the login. We may want to make this more complex, but it's not the main focus of this challenge.

## Run

1. `docker build -t challendar . --build-arg PASSWORD=H3110fr13nD`
2. `docker run -p 3000:3000 -p 4000:4000 challendar`
3. (change IP and port in script, start nc) `python3 solution.py`

## Concerns

1. ~~Payloads may be visible by other users although I have mitigated this already by disabling some of the CalDAV verbs. It may be necessary to regularly wipe or give different credentials.~~ Added regular 1min cleanup in `start.sh`.
2. Hopefully no one discovers a new vulnerability in Radicale or the `golang.org/x/net/webdav` package.