package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func TestAPI(t *testing.T) {
	t.Skip("Skip")

	var cookie string = "YW_DID=ciqui86rupojrarr1ag0; YW_SESS=HFmzbngWbfeJ73UpYSLRkUJxuCZQKV6ee7ti8zWAkAk"

	var targetUrl string = "https://datatracker.ietf.org/doc/html/rfc8949"

	authheaders := util.CtxHeader{}
	uuid := util.NewUUID()
	fmt.Printf("UUID: %s\n", uuid.String())

	http.Header(authheaders).Set("x-request-id", uuid.String())
	http.Header(authheaders).Set("cookie", cookie)

	ctx := gear.CtxWith[util.CtxHeader](context.Background(), &authheaders)
	sess, err := GetToken(ctx)
	require.NoError(t, err)

	http.Header(authheaders).Set("authorization", "Bearer "+sess.AccessToken)
	ctx = gear.CtxWith[util.CtxHeader](context.Background(), &authheaders)

	myGroups, err := ListMyGroups(ctx)
	require.NoError(t, err)

	gid := myGroups[0].ID
	fmt.Printf("GID: %s\n", gid.String())

	var cid util.ID
	cid, err = util.ParseID("cj1o30mnq8f3bcl8gaog")
	require.NoError(t, err)

	t.Run("CreateCreation", func(t *testing.T) {
		t.Skip("Skip")

		scrapingOutput, err := GetWeb(ctx, gid.String(), targetUrl)
		require.NoError(t, err)

		fmt.Printf("Scraping title: %s, length: %d\n", scrapingOutput.Title, len(scrapingOutput.Content))

		str, err := cbor.Diagnose(scrapingOutput.Content)
		fmt.Println(err, str)

		var doc content.DocumentNode
		err = cbor.Unmarshal([]byte(scrapingOutput.Content), &doc)
		if err != nil {
			panic(err)
		}

		creation, err := CreateCreation(ctx, &CreateCreationInput{
			GID:         *gid,
			Title:       scrapingOutput.Title,
			Content:     scrapingOutput.Content,
			Language:    "eng",
			OriginalUrl: &scrapingOutput.Url,
		})
		require.NoError(t, err)
		cid = creation.ID
		// cj1o30mnq8f3bcl8gaog, 364815
	})

	t.Run("Search", func(t *testing.T) {
		t.Skip("Skip")

		searchOutput, err := OriginalSearch(ctx, gid.String(), targetUrl)
		require.NoError(t, err)
		assert.Equal(t, 1, len(searchOutput.Hits))

		searchOutput, err = GroupSearch(ctx, "cbor", gid.String(), "")
		require.NoError(t, err)
		assert.Equal(t, 1, len(searchOutput.Hits))

		searchOutput, err = GroupSearch(ctx, "cbor", gid.String(), "eng")
		require.NoError(t, err)
		assert.Equal(t, 1, len(searchOutput.Hits))

		searchOutput, err = GroupSearch(ctx, "cbor", gid.String(), "zho")
		require.NoError(t, err)
		assert.Equal(t, 0, len(searchOutput.Hits))

		_, err = GroupSearch(ctx, "cbor", util.JARVIS.String(), "")
		assert.Error(t, err)

		creations, err := ListCreation(ctx, &GIDPagination{
			GID:    *gid,
			Fields: util.Ptr([]string{"title", "language"}),
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(creations))
	})

	t.Run("UpdateCreation", func(t *testing.T) {
		t.Skip("Skip")

		creation, err := GetCreation(ctx, gid.String(), cid.String())
		require.NoError(t, err)
		assert.True(t, len(*creation.Content) > 1000)

		update, err := UpdateCreation(ctx, &UpdateCreationInput{
			GID:       *gid,
			ID:        cid,
			UpdatedAt: *creation.UpdatedAt,
			Title:     util.Ptr(*creation.Title + " Updated"),
		})

		require.NoError(t, err)

		assert.True(t, *update.UpdatedAt > *creation.UpdatedAt)
	})

	t.Run("ReleaseCreation", func(t *testing.T) {
		t.Skip("Skip")
		res, err := ReleaseCreation(ctx, &CreatePublicationInput{
			GID:      *gid,
			CID:      cid,
			Language: "eng",
			Version:  1,
		})
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.True(t, len(res.Job) > 0)

		var publication *PublicationOutput

		i := 0
		for {
			time.Sleep(time.Second * 3)
			i++
			publication, err = GetPublicationByJob(ctx, res.Job)
			require.NoError(t, err)

			if publication.Version == 0 {
				continue
			}

			break
		}

		assert.True(t, len(*publication.Content) > 1000)
		assert.True(t, false)
	})

	t.Run("CreatePublication", func(t *testing.T) {
		t.Skip("Skip")
		res, err := ReleaseCreation(ctx, &CreatePublicationInput{
			GID:      *gid,
			CID:      cid,
			Language: "eng",
			Version:  1,
		})
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.True(t, len(res.Job) > 0)

		var publication *PublicationOutput

		i := 0
		for {
			time.Sleep(time.Second * 3)
			i++
			publication, err = GetPublicationByJob(ctx, res.Job)
			require.NoError(t, err)

			if publication.Version == 0 {
				continue
			}

			break
		}

		assert.True(t, len(*publication.Content) > 1000)
		assert.True(t, false)
	})
}
