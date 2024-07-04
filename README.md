# packer-plugin-ecloud

移动云packer插件

## 构建
```
## 基础镜像   
docker build -t  spinnaker/packer-build:go-20  .
     

## alpine二进制文件  
docker run --rm -v $(pwd):/build   spinnaker/packer-build:go-20    go build


```
