package api

import (
	"net/http"
	"path"
	"strconv"
	"webapp-server/db"
	"webapp-server/gpt"
	"webapp-server/lang"
	"webapp-server/obj"
	"webapp-server/router"
)

var Image = router.NewEndpointJson(
	"/api/image/:sessionHash/:chapter",
	false,
	func(request router.Request) (out interface{}, httpErr *obj.HTTPError) {
		var err error
		sessionHash := path.Base(request.R.URL.Path)
		chapterRaw := path.Base(path.Dir(request.R.URL.Path))
		var chapter uint64
		chapter, err = strconv.ParseUint(chapterRaw, 10, 32)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: lang.ErrorParsingRequest}
		}

		var session *obj.Session
		var game *obj.Game
		if session, err = db.GetSessionByHash(sessionHash); err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: lang.ErrorFailedLoadingGameData}
		}

		if game, err = db.GetGameByID(session.GameID); err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: lang.ErrorFailedLoadingGameData}
		}

		var image []byte
		image, err = db.GetImage(session.ID, uint(chapter))
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: lang.ErrorFailedLoadingGameData}
		}

		if image != nil || len(image) > 0 {
			return image, nil
		}

		var apiKey *string
		if apiKey, err = request.User.GetApiKey(session, game); err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusUnauthorized, Message: lang.ErrorNoValidKey}
		}
		if image, httpErr = gpt.GenerateImage(request.Ctx, *apiKey, "a red circle and a blue hand"); httpErr != nil {
			return nil, httpErr
		}

		if httpErr = db.SetImage(session.ID, uint(chapter), image); httpErr != nil {
			return nil, nil
		}

		return []byte{}, nil
	},
)
