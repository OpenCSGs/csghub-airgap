package image

import (
	"io"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/schollz/progressbar/v3"

	"csghub-airgap/internal/log"
)

// PullImage 拉取单个镜像，并显示进度和下载完成信息
func PullImage(image string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref)
	if err != nil {
		return nil, err
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	// 计算总大小
	var total int64
	for _, l := range layers {
		size, _ := l.Size()
		total += size
	}

	// 初始化进度条
	bar := progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription("Downloading "+image),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowIts(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() { println() }),
		progressbar.OptionSetRenderBlankState(true),
	)

	// 并发拉取层
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	progressChan := make(chan int64, 100)

	done := make(chan struct{})
	go func() {
		for n := range progressChan {
			_ = bar.Add64(n)
		}
		close(done)
	}()

	for _, layer := range layers {
		layer := layer
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rc, err := layer.Compressed()
			if err != nil {
				return
			}
			defer rc.Close()

			buf := make([]byte, 32*1024)
			for {
				n, err := rc.Read(buf)
				if n > 0 {
					progressChan <- int64(n)
				}
				if err != nil {
					if err != io.EOF {
						// 忽略其他错误
					}
					break
				}
			}
		}()
	}

	wg.Wait()
	close(progressChan)
	<-done
	_ = bar.Finish()

	// 使用 log 统一输出 INFO
	log.Infof("INFO[%s] File downloaded successfully: %s",
		time.Now().Format("2006-01-02T15:04:05-07:00"),
		image,
	)

	return img, nil
}

// PullImages 顺序拉取多个镜像，每个镜像独立进度条
func PullImages(images []string) ([]v1.Image, error) {
	var result []v1.Image
	for _, img := range images {
		imageObj, err := PullImage(img)
		if err != nil {
			return result, err
		}
		result = append(result, imageObj)
	}
	return result, nil
}
