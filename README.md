# Twitch Bot
By [UberSwe](https://uberswe.com)

This a project I started for fun to see how things work and if I could do it well. I probably spent way too much time on it and I should have gone for something simpler but here we are.

Run the project with `go run cmd/twitch/main.go`

Make sure to copy the `.env.example` and save it as `.env` adding your twitch client id and secret. You can find more info about this in the [Twitch API Documentation](https://dev.twitch.tv/docs/v5).

By default the project will listen at `http://localhost:8010` but you can configure this using parameters (to be documented).

If you go to this url you will be presented with a connect button which you can click to connect your channel account. Once connected you will get to the dashboard.

On the dashboard you will have a link to connect a bot, I recommend opening a private window or a window in a separate browser and logging into twitch with your bot account there and then following the link. Once connected you will now have a bot associated with your channel. This means you can have your own bot which is kind of the whole point of this project :).

Now, if you specify the `UNIVERSAL_BOT` and you set it equal to the name of your bot, for example `botbyuber` then this will be the default bot when someone connects their channel bot have not connected a bot. So you could allow others to user your bot, wohoo!

By connecting you will currently always be in your channels chat, this is temporary. Connecting your bot will also always keep it in chat, this is expected and will always be like this.

Check the Issues page to see what kind of known bugs exist. If you try this and find a bug consider opening an issue.

Contributions are welcome!