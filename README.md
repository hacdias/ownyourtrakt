# Own Your Trakt

Please use this at your **own risk**. Currently, I don't have a hosted version for general use.
If you're interested, you can host it yourself:

1. Create an application on https://trakt.tv/oauth/applications/new
2. Copy `.env.example` to `.env`
3. Fill the required fields. The `SESSION_KEY` is used to encrypt the session cookies. Please use a strong,
randomly generated, string.
4. `make`
5. Run the executable!
6. Go to `BASE_URL` and log in with your user.

I decided to keep the most information possible so your micropub endpoint can make whatever transformations
you want and you still have access to the IDs that can be used to fetch more info from the Trakt or other APIs.

## Shortcomings

1. It doesn't fetch watches that you add in the past.

## Example of episode request

```json
{
  "type": [
    "h-entry"
  ],
  "properties": {
    "published": [
      "2020-02-11T15:40:46Z"
    ],
    "watch-of": [
      {
        "properties": {
          "episode": [
            14
          ],
          "season": [
            3
          ],
          "title": [
            "A Slump, a Cross and Roadside Gravel"
          ],
          "trakt-id": [
            5611853642
          ],
          "url": [
            "https://trakt.tv/shows/young-sheldon/seasons/3/episodes/14"
          ],
          "ids": {
            "trakt": 3935937,
            "imdb": "tt11591958",
            "tmdb": 2072445,
            "tvdb": 7539973
          },
          "show": [
            {
              "properties": {
                "ids": {
                  "trakt": 119172,
                  "imdb": "tt6226232",
                  "tmdb": 71728,
                  "slug": "young-sheldon",
                  "tvdb": 328724
                },
                "title": [
                  "Young Sheldon"
                ],
                "url": [
                  "https://trakt.tv/shows/young-sheldon"
                ],
                "year": [
                  2017
                ]
              },
              "type": [
                "h-card"
              ]
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
