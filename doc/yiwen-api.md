# yiwen-api

## 读取网页

`GET https://api.yiwen.ltd/v1/scraping?url=encodedUrl`

示例：
```
GET https://api.yiwen.ltd/v1/scraping?url=https%3A%2F%2Fmp.weixin.qq.com%2Fs%2F6iCpGzsqnXcGZPoEhqJE4Q
Accept: application/json
Cookie: YW_DID=ciqui8xxxx; YW_SESS=HFmzbngWbxxxxxWAkAk
Authorization: Bearer hE2iAScESDIxxxxxxxeCMK
```

返回：
```json
{
  "retry": 0,
  "result": {
    "id": "cis0vcgcert0052hqvbg",
    "url": "https://mp.weixin.qq.com/s/6iCpGzsqnXcGZPoEhqJE4Q",
    "src": "",
    "title": "网络平台提供代币充值服务合规要点——以抖音抖币充值为例",
    "meta": {
      "og:article:author": "夏梦雅团队",
      "og:description": "本文共4803字,预计阅读时长为20分钟",
      "og:image": "https://mmbiz.qpic.cn/mmbiz_jpg/qjp6a5pznC2zfliaqxET0c8bia4gicA3BnMIb5dCNwGKTvTJPSpaicc4Ht920xH7MJlH9cLPA2RibcPyms1bjZ4g8GA/0?wx_fmt=jpeg",
      "og:site_name": "Weixin Official Accounts Platform",
      "og:type": "article",
      "og:url": "http://mp.weixin.qq.com/s?__biz=MzU2NzgzODEzMw==\u0026mid=2247485848\u0026idx=1\u0026sn=f7dd03a96e23d92dff462f754d528a70\u0026chksm=fc965d42cbe1d45473de15cb435468bb97d106f3d4f3a980fb9a0067aeee07ec0e80e1ab1449#rd"
    },
    "content": "WVqUuQACZHR5cGVjZG9jZ2Nvb...base64_url_raw_encode_cbor...5ZCI6KeE6KaB54K54oCU4oCU"
  }
}
```
