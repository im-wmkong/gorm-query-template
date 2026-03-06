package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"gorm.io/gorm/schema"
)

var (
	typeNames  = flag.String("type", "", "逗号分隔的类型名称列表；必须设置")
	output     = flag.String("output", "", "输出文件名；默认 srcdir/<type>_gen.go")
	moduleName = flag.String("module", "", "在导入中使用的模块名称；如果未设置，尝试从 go.mod 检测")
)

var namingStrategy = schema.NamingStrategy{
	SingularTable: true,
}

func main() {
	// 1. 解析参数
	flag.Parse()
	if len(*typeNames) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	goFile, outName := parseEnvAndArgs()

	// 2. 加载包
	pkgs := loadPackage(goFile)

	// 3. 查找目标包
	pkg := findTargetPackage(pkgs)

	// 4. 确定模块路径
	mod := determineModulePath(pkg)

	// 5. 初始化生成器
	g := Generator{
		pkgName:    pkg.Name,
		data:       make(map[string][]FieldInfo),
		outputPath: outName,
		modulePath: mod,
	}

	// 6. 处理类型
	g.processTypes(pkg)

	// 7. 生成代码
	g.generate()
}

func parseEnvAndArgs() (string, string) {
	// 解析当前目录下的文件或 GOFILE 环境变量指定的文件
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		log.Fatal("GOFILE 环境变量必须设置 (通过 go generate 运行)")
	}

	// 确定输出文件名
	outName := *output
	if outName == "" {
		baseName := strings.TrimSuffix(goFile, ".go")
		outName = baseName + "_gen.go"
	}
	return goFile, outName
}

func loadPackage(goFile string) []*packages.Package {
	absGoFile, err := filepath.Abs(goFile)
	if err != nil {
		log.Fatalf("无法获取 GOFILE 的绝对路径: %v", err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports | packages.NeedModule,
		Dir:  filepath.Dir(absGoFile), // 设置 Dir 为 GOFILE 所在目录
	}
	// 使用 file=pattern 语法来加载包含特定文件的包
	pkgs, err := packages.Load(cfg, "file="+absGoFile)
	if err != nil {
		log.Fatalf("加载包失败: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	return pkgs
}

func findTargetPackage(pkgs []*packages.Package) *packages.Package {
	typesList := strings.Split(*typeNames, ",")
	var pkg *packages.Package
	var foundPkg bool

	for _, p := range pkgs {
		for _, typeName := range typesList {
			if obj := p.Types.Scope().Lookup(typeName); obj != nil {
				pkg = p
				foundPkg = true
				break
			}
		}
		if foundPkg {
			break
		}
	}

	if !foundPkg {
		// 如果没找到包含目标类型的包，默认使用第一个包，或者报错
		if len(pkgs) > 0 {
			pkg = pkgs[0]
		} else {
			log.Fatal("未找到任何包")
		}
	}
	return pkg
}

func determineModulePath(pkg *packages.Package) string {
	mod := *moduleName
	if mod == "" {
		if pkg != nil && pkg.Module != nil {
			mod = pkg.Module.Path
		} else {
			// 如果无法从 packages 获取模块信息，尝试降级处理或报错
			log.Printf("警告: 无法检测模块路径，使用默认 'gorm-query-template'")
			mod = "gorm-query-template"
		}
	}
	return mod
}
