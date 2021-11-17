# Own Your Trakt

Please use this at your **own risk**. Currently, I don't have a hosted version for general use.
If you're interested, you can host it yourself:

1. Create an application on https://trakt.tv/oauth/applications/new
2. Copy `config.example.yaml` to `config.yaml`
3. Fill the required fields.
4. `go build`
5. Run the executable!

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
    "summary": [
      "Just watched: A Slump, a Cross and Roadside Gravel (Young Sheldon S3E14)"
    ],
    "watch-of": [
      {
        "type": [
          "h-cite"
        ],
        "properties": {
          "name": [
            "A Slump, a Cross and Roadside Gravel"
          ],
          "url": [
            "https://trakt.tv/shows/young-sheldon/seasons/3/episodes/14"
          ],
          "episode": [
            14
          ],
          "season": [
            3
          ],
          "trakt-watch-id": [
            5611853642
          ],
          "trakt-ids": {
            "trakt": 3935937,
            "imdb": "tt11591958",
            "tmdb": 2072445,
            "tvdb": 7539973
          },
          "episode-of": [
            {
              "type": [
                "h-cite"
              ],
              "properties": {
                "name": [
                  "Young Sheldon"
                ],
                "url": [
                  "https://trakt.tv/shows/young-sheldon"
                ],
                "published": [
                  2017
                ],
                "trakt-ids": {
                  "trakt": 119172,
                  "imdb": "tt6226232",
                  "tmdb": 71728,
                  "slug": "young-sheldon",
                  "tvdb": 328724
                }
              }
            }
          ]
        }
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
    "summary": [
      "Just watched: Maleficent: Mistress of Evil"
    ],
    "watch-of": [
      {
        "type": [
          "h-cite"
        ],
        "properties": {
          "name": [
            "Maleficent: Mistress of Evil"
          ],
          "url": [
            "https://trakt.tv/movies/maleficent-mistress-of-evil-2019"
          ],
          "published": [
            2019
          ],
          "trakt-watch-id": [
            5611853642
          ],
          "trakt-ids": {
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
