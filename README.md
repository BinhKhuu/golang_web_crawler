# 🕷️ Golang Web Crawler

A web crawler built with Go and React.

---

## 🛠️ Required Tools

### Core
| Tool | Version | Install |
|------|---------|---------|
| [Go](https://golang.org/dl/) | 1.25+ | `brew install go` |
| [Node.js](https://nodejs.org/) | 18+ | `brew install node` |
| [npm](https://www.npmjs.com/) | 9+ | Comes with Node.js |

### Infrastructure
| Tool | Version | Install |
|------|---------|---------|
| [Docker](https://docs.docker.com/get-docker/) | 24+ | `brew install --cask docker` |
| [Docker Compose](https://docs.docker.com/compose/) | 2+ | Included with Docker Desktop |

### Database
| Tool | Version | Install |
|------|---------|---------|
| [golang-migrate](https://github.com/golang-migrate/migrate) | latest | `brew install golang-migrate` |

### Environment
| Tool | Version | Install |
|------|---------|---------|
| [godotenv](https://github.com/lpernett/godotenv) | latest | `go get github.com/lpernett/godotenv` |

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/your-username/golangwebcrawler.git
cd golangwebcrawler
```


### playwright package
playwright is used to 'smart crawl' installation requires installing all the playwright depdencides

```
go get github.com/playwright-community/playwright-go

go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps

```

* Drivers Playwright Go driver v1.57.0 installs ( version number is a sample )
