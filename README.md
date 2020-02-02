# Own Your Trakt

Still in the ♨️ oven. Currently, I don't have a hosted version for general use. If you're interested,
you can host it yourself:

1. Create an application on https://trakt.tv/oauth/applications/new
2. Copy `.env.example` to `.env`
3. Fill the required fields. The `SESSION_KEY` is used to encrypt the session cookies. Please use a strong,
randomly generated, string.
4. `go build`
5. Run the executable!
6. Go to `BASE_URL` and log in with your user.
