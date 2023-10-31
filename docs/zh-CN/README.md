# 关于 Airway

Airway 是使用 go 语言开发的全栈 Web 框架，受到 Ruby on Rails 框架的启发。

在开发 `Airway` 之前，作者有 10 年以上 Rails 开发经验，对于 Rails 框架的优点和缺点，有比较深入的理解。

Rails 框架的优点之一，在于清晰固定的目录结构。这样的好处是，对于接触一个新的 Rails 项目，可以很快的上手。

Rails 框架的优点其二，在于【约定胜于配置】的设计思想，其默认的规则，往往就满足了开发需求，而不需要写大量的配置文件。

Rails 框架的缺点，有以下几点：

1. 缺少业务逻辑层抽象，导致业务逻辑散落在控制器和 Model 之间。对于复杂的项目，你也许能看到几百行的 Model 定义。
2. 不够模块化。
3. 有些设计，属于黑魔法，对于第一次接触的人来说，有点难以掌握。

另外，由于 Rails 是基于 Ruby 语言开发的，其命令行工具，在生成框架代码时，运行速度非常慢（相比于 go 来说）。

所以，在吸取了 Rails 框架的优点，结合 Go 语言的特性，针对性的改进 Rails 框架的缺点，就这样，我开发出了 **`Airway`**。

# 快速上手

提示：目前 `airway` 还处于开发阶段，以下内容仅适用于当前的版本。如果有不完善的地方，或者好的点子和改进意见，欢迎提 issue。

## 1. 创建目录结构

我们使用`gonew`工具，来创建新的 go 项目：

```bash
$ gonew github.com/daqing/airway example.com/foo/bar
```

针对你的项目情况，替换 `example.com/foo/bar` 为真实的模块名称。

## 2. 搭建开发环境

#### 2.1

首先，项目使用 [just](https://github.com/casey/just) 来执行一些脚本命令。

关于如何安装 `just` 请参考其中文文档：[https://github.com/casey/just/blob/master/README.中文.md](https://github.com/casey/just/blob/master/README.中文.md)

除了 just, 项目还用到了以下软件：

- air: [github.com/cosmtrek/air](https://github.com/cosmtrek/air) 热重载 Go 应用的工具

- overmind: [github.com/DarthSim/overmind](https://github.com/DarthSim/overmind) Process manager for Procfile-based applications and tmux

- bun: [https://bun.sh](https://bun.sh) Bun is a fast JavaScript
all-in-one toolkit

对于 macOS 系统，只需要执行:

```bash
$ just setup
```

就可以把上述依赖的软件安装好。


#### 2.2

其次，项目使用了 `.env` 作为配置。

需要创建 `.env.local` 文件：

```bash
$ cp .env.example .env.local
```

这个文件，定义了几个环境变量，说明如下：

**AIRWAY_PG_URL**

连接PostgreSQL字符串，类似这样的形式:

`postgres://daqing:passwd@127.0.0.1:5432/airway`

**AIRWAY_PORT**

服务器监听的端口，默认为 `"1900"`

**AIRWAY_STORAGE_DIR**

存储用户上传文件的绝对路径

例如: `/var/www/storage`（不要以'/'结尾）

**AIRWAY_ASSET_HOST**

静态资源的托管域名（可用于配置CDN）

例如: `https://assets.example.com"`

**AIRWAY_PWD**

项目所在的绝对路径（不要以'/'结尾）

例如: `/Users/joe/projects/airway`