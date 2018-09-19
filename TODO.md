4) change resource flag so it can be specified multiple times or take a list
1) discover the correct profile from the instance itself
1) TEST!
1) review struct names: currently named so they're exported, should not be
1) review flag construction. it's been suggested that it either shouldn't be in init, or flag declarations should also be in init
1) continue to remove os.Exit and log.Fatal from everything but main, and maybe? from main.
1) Oh shit this would be way better: http://eagain.net/articles/go-dynamic-json/. Also, this: https://mholt.github.io/json-to-go/
