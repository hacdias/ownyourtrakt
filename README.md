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

I decided to keep the most information possible so your micropub endpoint can make whatever transformations
you want and you still have access to the IDs that can be used to fetch more info from the Trakt or other APIs.

## Example of episode request

```json
{
  "type": [
    "h-entry"
  ],
  "properties": {
    "published": [
      "2020-01-17T19:46:33Z"
    ],
    "watch-of": [
      {
        "properties": {
          "title": [
            "Episode 2"
          ],
          "type": [
            "episode"
          ],
          "url": [
            "https://trakt.tv/shows/sex-education/seasons/2/episodes/2"
          ],
          "episode": [
            2
          ],
          "season": [
            2
          ],
          "ids": {
            "trakt": 3855303,
            "imdb": "tt9699190",
            "tmdb": 1994667,
            "tvdb": 7480886
          },
          "show": [
            {
              "type": [
                "h-card"
              ],
              "properties": {
                "title": [
                  "Sex Education"
                ],
                "url": [
                  "https://trakt.tv/shows/sex-education"
                ],
                "year": [
                  2019
                ],
                "ids": {
                  "trakt": 140590,
                  "imdb": "tt7767422",
                  "tmdb": 81356,
                  "slug": "sex-education",
                  "tvdb": 356317
                }
              }
            }
          ]
        },
        "type": [
          "h-card"
        ]
      }
    ]
  }
}
```

## Example of movie request

```json
{
  "type": [
    "h-entry"
  ],
  "properties": {
    "published": [
      "2020-01-17T22:31:25Z"
    ],
    "watch-of": [
      {
        "type": [
          "h-card"
        ],
        "properties": {
          "title": [
            "Maleficent: Mistress of Evil"
          ],
          "type": [
            "movie"
          ],
          "url": [
            "https://trakt.tv/movies/maleficent-mistress-of-evil-2019"
          ],
          "year": [
            2019
          ],
          "ids": {
            "trakt": 265465,
            "imdb": "tt4777008",
            "tmdb": 420809,
            "slug": "maleficent-mistress-of-evil-2019"
          }
        }
      }
    ]
  }
}
```