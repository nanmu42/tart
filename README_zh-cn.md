[English](https://github.com/nanmu42/tart/blob/master/README.md) | **中文**

# 蛋挞（Tart）

[![GoDoc](https://godoc.org/github.com/nanmu42/tart?status.svg)](https://godoc.org/github.com/nanmu42/tart)
[![Build status](https://jihulab.com/nanmu42/tart/badges/main/pipeline.svg)](https://jihulab.com/nanmu42/tart/-/pipelines)

<div align="center">
  <img width="360" src="https://user-images.githubusercontent.com/8143068/197567829-b2d9783d-a660-41c6-bea4-5945dfa1ccb3.png">
</div>


蛋挞是一个教学目的，非官方的Gitlab Runner，通过简明地实现Gitlab Runner功能的一个子集，展示Gitlab Runner的设计和实现方法。

举个例子，蛋挞可以[运行自己的CI job，运行自己的测试和编译自己](https://jihulab.com/nanmu42/tart/-/jobs/4980020)。

特色：

* 折腾；
* 使用[Firecracker](https://firecracker-microvm.github.io/)和`/dev/kvm`，让每个job在一个**两秒内**启动的虚拟机中运行，我目前没在公开资料里查到这么做的；
* 代码量少，大概2000行实现了Gitlab Runner的核心功能：job的获取、执行、环境隔离、日志和结果的上报；
* 在每个星期四运行job会有特殊效果。

只实现了核心功能，产物上传、service这些功能是不支持的。换句话说，不要用于生产环境（真的会有人这么做吗）。

## 相关文章

在写了在写了，咕咕咕 `_(:зゝ∠)_`

## 使用方法

蛋挞需要在可以访问`/dev/kvm`的Linux环境下运行：

```bash
sudo setfacl -m u:${USER}:rw /dev/kvm
```

1. 从release页面下载蛋挞和Firecracker的二进制，并将它们置于`$PATH`
2. 从release页面下载RootFS和Linux内核，把它们放到工作文件夹，比如`~/tart`
3. 为tart创建的虚拟机预先配置网络，请参考`rootfs/setup-tuntap.sh`
4. cd到工作文件夹
5. 注册tart为你项目的Gitlab Runner：`tart register --endpoint https://gitlab.example.com --token your_token_here > tart.toml`
6. 启动tart：`tart run`
7. 在Gitlab上触发CI，为了确保job会调度到tart上，你可能得禁用项目的shared runner
8. 观看tart工作（或者爆炸）

## 编译方式

```bash
make
```

产物在`bin`文件夹中。

虚拟机的RootFS和Linux内核编译请参考`rootfs`文件夹。

## 为啥叫蛋挞？

我喜欢吃蛋挞。

## 许可证

MIT

请自由享受和贡献开源。

第三方项目许可证请参阅`THIRD_PARTY_LICENSES.md`.

logo的照片来自于Ashley Byrd on Unsplash，Gopher在[gopherize.me](https://gopherize.me/)生成。