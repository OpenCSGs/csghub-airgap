package image

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"os"
	"time"
)

func Import(spec *spec.ImageSpec) error {
	imageFilePath := spec.ImageFilePath()
	imageFileName := spec.ImageFileName()

	if _, err := os.Stat(imageFilePath); err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Images import file %s does not exist", imageFileName)
			return nil
		}
		return err
	}

	log.Infof("Start importing images from %s", imageFileName)

	// ---------- Load ----------
	loadStart := time.Now()

	//imgMap, err := crane.Load(imageFilePath)
	//if err != nil {
	//	return err
	//}
	//
	//// Push 到目标仓库
	//for imgName, img := range imgMap {
	//	log.Infof("Importing image: %s", imgName)
	//	if err := crane.Push(img, imgName); err != nil {
	//		return err
	//	}
	//}

	log.Infof("Imported images from %s (elapsed %.3fs)", imageFileName, time.Since(loadStart).Seconds())
	return nil
}
