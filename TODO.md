4) change resource flag so it can be specified multiple times or take a list
1) discover the correct profile from the instance itself
1) TEST!
1) review struct names: currently named so they're exported, should not be
1) review flag construction. it's been suggested that it either shouldn't be in init, or flag declarations should also be in init
1) continue to remove os.Exit and log.Fatal from everything but main, and maybe? from main.
1) Oh shit this would be way better: http://eagain.net/articles/go-dynamic-json/. Also, this: https://mholt.github.io/json-to-go/
1) handle regions set by terraform rather than by profile
  * This can be done by using credentials.NewSharedCredentials with the [profile name and an explicit region](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html)
  * will need to change getProfilesFromFile to return structs that have profiles and region
