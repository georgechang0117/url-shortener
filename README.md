# url-shortener

An implementation of URL shortener using Go programming language.

## API Example:

```
# Upload URL API
curl -X POST -H "Content-Type:application/json" http://localhost/api/v1/urls -d '{
"url": "https://www.google.com/",
"expireAt": "2021-07-11T09:20:41Z"
}'
# Response
{
  "id":"YbWE4pOZCTH",
  "shortUrl":"http://localhost/YbWE4pOZCTH"
}
# ------------------
# Redirect URL API
curl -L -X GET http://localhost/YbWE4pOZCTH => REDIRECT to original URL
```

## Up and Running

Use docker-compose to run all services (url-shortener, mysql and redis)

```bash
cd docker
docker-compose up
```

## Project Structure

```
├── base
│   ├── base62
│   ├── cache
│   └── lock
├── core
│   ├── dao
│   └── urlshortener
├── docker
├── main
└── rest
```

- **base**: 實作商業邏輯會用到的基本工具
- **core**: 商業邏輯實作
- **docker**: docker-compose 相關檔案
- **main**: main folder
- **rest**: Web API 相關實作

## Infrastructure

- MySQL: persistent store，存放短網址資料
- Redis: cache store，緩存短網址資料，優化存取效能

使用原因：相對熟悉，實作起來快很多

## Libraries

- redis-lock: 用來實作 distributed lock
- gorm: ORM，用來實作 data access object
- echo: Web framework，實作 Rest APIs
- go playground validator: API validation
- redis: 實作 redis remote cache
- zap: logging 使用
- clock: 單元測試時間相關邏輯使用

## Testing

### Run all tests

```bash
go test $(go list github.com/georgechang0117/url-shortener/...)
```

### Generate mocks for golang interfaces

Use [mockery](https://github.com/vektra/mockery)

```bash
mockery --name URLShortener
```

## Implementation Details

實作細節與思路

- 使用 base62 + rand uint64 產生 url_id，一方面能產生 url safe 的 id，另一方面也能讓 key 的長相跟產生順序無關 (unpredictable key)
- 將資料存取的邏輯全收在 core/urlshortener package，這樣做的好處是 urlshortener 的使用者 rest handler 只要負責從 urlshortener 的 API 上傳資料與拿到資料即可
- 若同時間有大量 redirect request，但是 cache miss 的話，壓力就會送往後端的 db，造成 cache stampede。因此在 core/urlshortener 加入 distributed lock 解決這個問題，同時間只有一個 request 能夠存取 db 更新 cache，其他同時間的 request 便能直接從 cache 取得資料
- 若使用者輸入不存在的 url_id 的話，自然會 cache miss 再轉進後端 db 尋找，造成後端 db 壓力，採取的作法是若從後端 db 找不到就 cache empty data，並設定時效很短的 TTL。這樣短時間內存取相同的網址時，便能直接從 cache 找到資料回應，設定較短的 TTL 是避免 empty data 的資料在 cache 存放太久佔用 memory。不過此招只能防君子，若使用者得知 url_id 的驗證規則，並製造大量隨機 url_id 的惡意攻擊，還是會對後端 db 造成影響
- Upload URL API 加上了 url 必須為 uri 格式的驗證、expireAt 必須為 RFC3339 格式驗證、expireAt 時間必須大於現在時間驗證
- Redirect API 加上了 url_id 長度驗證、url_id 必須為 alphabet+num 格式驗證，若驗證不過直接回應 404 status error，避免揭露規則

## TODOs

- 使用 bloom filter 過濾 redirect API，利用 bloom filter 特性判斷 url_id 是否存在，若不存在就直接擋掉，才能防止產生隨機 url_id 的惡意攻擊造成 cache penetration
- 以目前的規格不需考慮更新問題，可在存取 remote cache 之前多加一層 local cache
- 規格不需要 transaction 與 table join 的話，db 改用 Mongo 效能應該可以更好
- 加上 api rate limit，防止惡意攻擊
