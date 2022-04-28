# go-amputate
Go library for calling the Amputatorbot API (https://www.amputatorbot.com/)

## Usage

```go
package main

import (
    goamputate "github.com/tyzbit/go-amputate"
)

var bot goamputate.AmputatorBot

func main() {
    options := map[string]string{}
    // These have defaults, you don't have to set them
    options["gac"] = "true"
    options["md"] = "3"

    urls := []string{}
    urls[0] = "https://www.google.com/amp/s/electrek.co/2018/06/19/tesla-model-3-assembly-line-inside-tent-elon-musk/amp/"
    urls[1] = "https://amp.cnn.com/cnn/2022/01/03/tech/elizabeth-holmes-verdict/index.html"

    amputatedLinks, err := bot.Amputate(urls, options)
    if err != nil {
        // Handle the error however you wish
        return
    }

    if len(amputatedLinks) == 0 {
        // Handle receiving no URLs back however you wish
        return
    }

    // At this point amputatedLinks has at least one amputated url
    fmt.Println(strings.Join(amputatedLinks, ","))
}
```