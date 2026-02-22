# Zenn Topics

[zenn.dev](https://zenn.dev) のトピックを検索する CLI

## 使用方法

```console
Usage:
  zenn-topics [options] <query>

Options:
  -h, --help      Show help
      --refresh   Bypass cache and refresh topics
  -v, --verbose   Print verbose logs to stderr
  -V, --version   Show version
```

## クイックスタート

```sh
# インストール
go install github.com/takymt/zenn-topics@latest

# 検索
zenn-topics <query>
```

ビルド済みのバイナリについては [GitHub Releases](https://github.com/takymt/zenn-topics/releases) からダウンロードできます

## 作成者

@takymt (a.k.a tarte)

## ライセンス

MIT