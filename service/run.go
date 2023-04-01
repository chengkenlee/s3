package service

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
	"s3/util"
	"strconv"
	"strings"
	"time"
)

var (
	s3filename    string
	bucket        *string
	key           *string
	accessKey     string
	secretKey     string
	endPoint      string
	myACL         *string
	metadataKey   string
	metadataValue string
	myMetadata    map[string]*string
	s3Config      *aws.Config
	newSession    *session.Session
)

func inits() {
	if len(util.P.UploadFile) != 0 {
		s3filename = util.P.UploadFile
	}
	if len(util.P.DownloadFile) != 0 {
		s3filename = util.P.DownloadFile
	}
	if len(util.P.DeleteFile) != 0 {
		s3filename = util.P.DeleteFile
	}
	bucket = aws.String(util.Config.GetString("ecs.s3.bucket"))                             //bucket名称
	key = aws.String(fmt.Sprintf("%s/%s", util.Config.GetString("ecs.s3.key"), s3filename)) //object keyname
	accessKey = util.Config.GetString("ecs.s3.access.key")
	secretKey = util.AesDecrypt(util.Config.GetString("ecs.s3.secret.key"), util.ENCKEY)
	endPoint = util.Config.GetString("ecs.s3.endpoint")     //endpoint设置，不要动
	myACL = aws.String(util.Config.GetString("ecs.s3.acl")) //acl 设置
	metadataKey = ""                                        //自定义Metadata key
	metadataValue = ""                                      //自定义Metadata value
	myMetadata = map[string]*string{
		metadataKey: &metadataValue,
	}
	// Configure to use S3 Server
	s3Config = &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String(endPoint),
		Region:           aws.String(util.Config.GetString("ecs.s3.region")),
		DisableSSL:       aws.Bool(util.Config.GetBool("ecs.s3.ssl")),
		S3ForcePathStyle: aws.Bool(util.Config.GetBool("ecs.s3.virtual-host")), //virtual-host style方式，不要修改
	}
	newSession = session.Must(session.NewSession(s3Config))
	s3Client := s3.New(newSession)
	cparams := &s3.HeadBucketInput{
		Bucket: bucket, // Required
	}
	_, err := s3Client.HeadBucket(cparams)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

/*获取某个文件大小*/
func getsize(file string) int64 {
	svc := s3.New(newSession)
	params := &s3.ListObjectsInput{
		Bucket: bucket,
	}
	resp, err := svc.ListObjects(params)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	name := fmt.Sprintf("s3-package/sams/%s", file)
	for _, item := range resp.Contents {
		if name == *item.Key {
			return *item.Size
		}
	}
	return 0
}

/*删除文件*/
func deletes() {
	svc := s3.New(newSession)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: bucket, Key: key})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(fmt.Sprintf("delete %s done!", fmt.Sprintf("%s/s3-package/sams/%s", endPoint, s3filename)))
}

/*下载文件*/
func download() {
	file, err := os.Create(util.P.DownloadFile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer file.Close()

	/*进度条*/
	/*start*/
	var doneC = make(chan int)
	Size := getsize(util.P.DownloadFile)
	totalSizeM := fmt.Sprintf("%.2f", float64(Size)/1024/1024) + "M"
	// 进度条
	tic := time.Tick(1 * time.Second)
	go func(c chan int) {
		for {
			select {
			case <-doneC:
				return
			case <-tic:
				info, e := os.Stat(util.P.DownloadFile)
				if e != nil {
					fmt.Println(fmt.Sprintf("get file:%s size err:%v, sleep 10 second and try again.", util.P.DownloadFile, e))
					time.Sleep(5 * time.Second)
					continue
				}
				currSize := info.Size()
				currSizeM := fmt.Sprintf("%.2f", float64(currSize)/1024/1024) + "M"
				processRate := float64(currSize) / float64(Size) * 100
				rate := strconv.FormatFloat(processRate, 'f', 2, 64)
				fmt.Println(fmt.Sprintf("%s (%s/%s)%s", fmt.Sprintf("__s3_File_DownLoad[%s]", util.P.DownloadFile), currSizeM, totalSizeM, strings.Repeat("#", int(processRate/2))+rate+"%"))
				if currSize >= Size {
					return
				}
			}
		}
	}(doneC)
	/*end*/

	downloader := s3manager.NewDownloader(newSession)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
		})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(fmt.Sprintf("%s (%s/%s)%s", fmt.Sprintf("__s3_File_DownLoad[%s]", util.P.DownloadFile), totalSizeM, totalSizeM, strings.Repeat("#", 50)+"100%"))
	fmt.Println(file.Name(), "下载完成!")
}

/*上传文件*/
func upload() {
	// Upload the file to S3.
	uploader := s3manager.NewUploader(newSession)
	f, err := os.Open(util.P.UploadFile)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	/*进度条*/
	lost, err := os.Stat(util.P.UploadFile)
	/*start*/
	var doneC = make(chan int)
	totalSizeM := fmt.Sprintf("%.2f", float64(lost.Size())/1024/1024) + "M"
	// 进度条
	tic := time.Tick(3 * time.Second)
	go func(c chan int) {
		for {
			select {
			case <-doneC:
				return
			case <-tic:
				Size := getsize(util.P.UploadFile)
				currSize := Size
				currSizeM := fmt.Sprintf("%.2f", float64(currSize)/1024/1024) + "M"
				processRate := float64(currSize) / float64(lost.Size()) * 100
				rate := strconv.FormatFloat(processRate, 'f', 2, 64)
				fmt.Println(fmt.Sprintf("%s (%s/%s)%s", fmt.Sprintf("__s3_File_Upload[%s]", util.P.UploadFile), currSizeM, totalSizeM, strings.Repeat("#", int(processRate/2))+rate+"%"))
				if currSize >= lost.Size() {
					return
				}
			}
		}
	}(doneC)
	/*end*/

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:   bucket,
		Key:      key,
		Body:     f,
		ACL:      myACL,
		Metadata: myMetadata,
	}, func(u *s3manager.Uploader) {
		u.PartSize = 200 * 1024 * 1024 * 1024 // 分块大小,当文件体积超过200G开始进行分块上传
		u.LeavePartsOnError = true
		u.Concurrency = 3
	}) //并发数
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(fmt.Sprintf("%s (%s/%s)%s", fmt.Sprintf("__s3_File_Upload[%s]", util.P.UploadFile), totalSizeM, totalSizeM, strings.Repeat("#", 50)+"100%"))
	fmt.Println(result.Location, "文件上传成功!")
}

/*展示列表*/
func list() {
	// bucket后跟，go run ....go bucketname
	svc := s3.New(newSession)
	params := &s3.ListObjectsInput{
		Bucket: bucket,
	}
	resp, err := svc.ListObjects(params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(endPoint)
	fmt.Println(fmt.Sprintf("%-10s%-85s%-40s(%s)%-20s%-10s", "Count", "Name", "Last modified", "KMGTPE", "Size", "Storage class"))
	for i, item := range resp.Contents {
		fmt.Println(fmt.Sprintf("%-10d%-85s%-40s(%s)%-20d%-10s", i, *item.Key, *item.LastModified, bytec(*item.Size), *item.Size, *item.StorageClass))
	}
	return
}

// 以1024作为基数
func bytec(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func Run() {
	inits()
	if util.P.List && len(util.P.UploadFile) == 0 {
		list()
		return
	}
	if len(util.P.UploadFile) != 0 {
		upload()
		return
	}
	if len(util.P.DownloadFile) != 0 {
		download()
		return
	}
	if len(util.P.DeleteFile) != 0 {
		deletes()
		return
	}
}
