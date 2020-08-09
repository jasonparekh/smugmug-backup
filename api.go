package smugmug

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type requestsHandler interface {
	get(string, interface{}) error
}

// userAlbums returns the list of albums belonging to the suer
func (w *Worker) userAlbums() ([]album, error) {
	uri := w.userAlbumsURI()
	return w.albums(uri)
}

// userAlbumsURI returns the URI of the first page of the user albums. It's intended to be used
// as argument for a call to albums()
func (w *Worker) userAlbumsURI() string {
	var u user
	path := fmt.Sprintf("/api/v2/user/%s", w.cfg.Username)
	w.req.get(path, &u)
	return u.Response.User.Uris.UserAlbums.URI
}

// albums make multiple calls to obtain the full list of user albums. It calls the albums endpoint
// unless the "NextPage" value in the response is empty
func (w *Worker) albums(firstURI string) ([]album, error) {
	uri := firstURI
	var albums []album
	for uri != "" {
		var a albumsResponse
		if err := w.req.get(uri, &a); err != nil {
			return albums, fmt.Errorf("Error getting albums from %s. Error: %v", uri, err)
		}
		albums = append(albums, a.Response.Album...)
		uri = a.Response.Pages.NextPage
	}
	return albums, nil
}

// albumImages make multiple calls to obtain all images of an album. It calls the album images
// endpoint unless the "NextPage" value in the response is empty
func (w *Worker) albumImages(firstURI string, albumPath string) ([]albumImage, error) {
	uri := firstURI
	var images []albumImage
	for uri != "" {
		var a albumImagesResponse
		if err := w.req.get(uri, &a); err != nil {
			return images, fmt.Errorf("Error getting album images from %s. Error: %v", uri, err)
		}
		// Loop over response in inject the albumPath and then append to the images
		for _, i := range a.Response.AlbumImage {
			i.AlbumPath = albumPath
			images = append(images, i)
		}
		uri = a.Response.Pages.NextPage
	}

	return images, nil
}

// saveImages calls saveImage or saveVideo to save a list of album images to the given folder
func (w *Worker) saveImages(images []albumImage, folder string) {
	for _, image := range images {
		if image.IsVideo {
			if err := w.saveVideo(image, folder); err != nil {
				log.Warnf("Error: %v", err)
			}
			continue
		}
		if err := w.saveImage(image, folder); err != nil {
			log.Warnf("Error: %v", err)
		}
	}
}

// saveImage saves an image to the given folder unless its name is empty
func (w *Worker) saveImage(image albumImage, folder string) error {
	if image.Name() == "" {
		return errors.New("Unable to find valid image filename, skipping..")
	}
	dest := fmt.Sprintf("%s/%s", folder, image.Name())
	log.Debug(image.ArchivedUri)
	return w.downloadFn(dest, image.ArchivedUri, image.ArchivedSize)
}

// saveVideo saves a video to the given folder unless its name is empty od is still under processing
func (w *Worker) saveVideo(image albumImage, folder string) error {
	if image.Processing { // Skip videos if under processing
		return fmt.Errorf("Skipping video %s because under processing, %#v\n", image.Name(), image)
	}

	if image.Name() == "" {
		return errors.New("Unable to find valid video filename, skipping..")
	}
	dest := fmt.Sprintf("%s/%s", folder, image.Name())

	var v albumVideo
	log.Debug("Getting ", image.Uris.LargestVideo.Uri)
	if err := w.req.get(image.Uris.LargestVideo.Uri, &v); err != nil {
		return fmt.Errorf("Cannot get URI for video %+v. Error: %v", image, err)
	}

	return w.downloadFn(dest, v.Response.LargestVideo.Url, v.Response.LargestVideo.Size)
}