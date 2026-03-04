package bucket

import "fmt"

type S3Conf struct {
	Region          string `yaml:"region"`   // aws: "eu-central-1", hetzner: "hel1" (or any; used for signing)
	Endpoint        string `yaml:"endpoint"` // hetzner: "https://<name>.hel1.your-objectstorage.com", aws: "" (empty)
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Prefix          string `yaml:"prefix"`         // optional: "dev/"
	UsePathStyle    bool   `yaml:"use_path_style"` // hetzner usually false; minio often true
}

func (c *S3Conf) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("s3.bucket is required")
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return fmt.Errorf("s3 access keys are required")
	}
	if c.Region == "" {
		return fmt.Errorf("s3.region is required")
	}
	// Endpoint empty => AWS default resolver.
	// Endpoint set => S3-compatible (Hetzner/MinIO/etc).
	return nil
}
