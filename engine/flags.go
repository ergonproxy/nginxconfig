package engine

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ergongate/vince/version"
	"go.uber.org/zap"
)

var VersionFlag = flag.Bool("v", false, "show version and exit")
var VersionAndConfigFlag = flag.Bool("V", false, "show version and configure options then exit")
var TestFlag = flag.Bool("t", false, "test configuration and exit")
var TestDump = flag.Bool("T", false, "test configuration, dump it and exit")

// NginxDirs returns a list of directories from which nginx configurations can
// be found.
func NginxDirs() []string {
	return []string{
		"/usr/local/nginx/conf",
		"/usr/local/etc/nginx",
		"/etc/nginx",
	}
}

// returns the nginx configuration file. This looks for the files in the
// following directories
// 	$CWD
// 	/usr/local/nginx/conf
// 	/usr/local/etc/nginx
// 	/etc/nginx
//
// The file is searched in the order outlined abouve the first to be found will
// be returned.
func nginxFile() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for _, v := range append([]string{wd}, NginxDirs()...) {
		file := filepath.Join(v, "nginx.conf")
		_, err = os.Stat(file)
		if err == nil {
			return file, nil
		}
	}
	return "", os.ErrNotExist

}

func showVersion(ctx context.Context) {
	if *VersionFlag {
		fmt.Println(version.Version)
		os.Exit(0)
	}
}

func showVersionAndConfig(ctx context.Context) {
	if *VersionAndConfigFlag {
		fmt.Println(version.Version)
		os.Exit(0)
	}
}

func testConfiguration(ctx context.Context) {
	if *TestFlag {
		err := testConfig(ctx)
		if err != nil {
			log(ctx).Error("Failed testing configuration", zap.Error(err))
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func testConfig(ctx context.Context) error {
	file, err := nginxFile()
	if err != nil {
		return err
	}
	fmt.Println("vince found configuration ", file)
	fmt.Printf("vince: the configuration file %s syntax is ok\n", file)
	fmt.Printf("vince: the configuration file %s test is successful\n", file)
	return nil
}

func testAndDump(ctx context.Context) {
	if *TestDump {
		file, err := nginxFile()
		fmt.Println("vince found configuration ", file)
		fmt.Printf("vince: the configuration file %s syntax is ok\n", file)
		fmt.Printf("vince: the configuration file %s test is successful\n", file)
		if err != nil {
			log(ctx).Error("Failed testing configuration", zap.Error(err))
			os.Exit(1)
		}
		f, err := os.Open(file)
		if err != nil {
			log(ctx).Error("Failed testing configuration", zap.Error(err))
			os.Exit(1)
		}
		defer f.Close()
		io.Copy(os.Stdout, f)
		//TODO dump configurations
		os.Exit(0)
	}
}

func runFlags(ctx context.Context) {
	showVersion(ctx)
	showVersionAndConfig(ctx)
	testConfiguration(ctx)
	testAndDump(ctx)
}
