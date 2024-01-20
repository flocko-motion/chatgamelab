package api

import (
	"net/http"
	"path"
	"strconv"
	"time"
	"webapp-server/db"
	"webapp-server/lang"
	"webapp-server/obj"
	"webapp-server/router"
)

var Image = router.NewEndpoint(
	"/api/image/",
	true,
	"image/png",
	func(request router.Request) (out interface{}, httpErr *obj.HTTPError) {
		var err error
		chapterRaw := path.Base(request.R.URL.Path)
		sessionHash := path.Base(path.Dir(request.R.URL.Path))
		var chapterId uint64
		chapterId, err = strconv.ParseUint(chapterRaw, 10, 32)
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: http.StatusBadRequest, Message: lang.ErrorParsingRequest}
		}

		var session *obj.Session
		if session, err = db.GetSessionByHash(sessionHash); err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: lang.ErrorFailedLoadingGameData}
		}

		// image creation can take a while - so we query the db until it's ready, or timeout
		for i := 0; i < 20; i++ {
			var chapter *obj.Chapter
			chapter, err = db.GetChapter(session.ID, uint(chapterId))
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: lang.ErrorFailedLoadingGameData}
			}

			if chapter.Image != nil && len(chapter.Image) > 0 {
				return chapter.Image, nil
			}

			time.Sleep(1 * time.Second)
		}

		return nil, &obj.HTTPError{StatusCode: 404, Message: "Image not found"}
	},
)
