# DivMinder API Documentation

## 개요

DivMinder API는 YieldMax ETF의 배당 정보를 제공하는 RESTful JSON API입니다. 모든 데이터는 매일 자동으로 업데이트되며, GitHub Pages를 통해 제공됩니다.

## Base URL

```
https://your-username.github.io/divminder-crawler
```

## 인증

API는 공개적으로 접근 가능하며 인증이 필요하지 않습니다.

## Rate Limiting

GitHub Pages의 기본 제한사항을 따릅니다. 과도한 요청은 피해주세요.

## CORS

모든 엔드포인트는 CORS를 지원하여 웹 애플리케이션에서 직접 호출할 수 있습니다.

## 엔드포인트

### 1. ETF 목록

#### `GET /etfs.json`

전체 YieldMax ETF 목록을 반환합니다.

**응답 예시:**
```json
[
  {
    "symbol": "TSLY",
    "name": "YieldMax™ TSLA Option Income Strategy ETF",
    "group": "GroupA",
    "frequency": "weekly"
  }
]
```

**필드 설명:**
- `symbol`: ETF 심볼
- `name`: ETF 전체 이름
- `group`: 배당 그룹 (GroupA, GroupB, GroupC, GroupD, Weekly, Target12)
- `frequency`: 배당 빈도

### 2. 메타데이터 포함 ETF 목록

#### `GET /etfs_enriched.json`

Alpha Vantage API에서 수집한 메타데이터가 포함된 ETF 목록을 반환합니다.

**응답 예시:**
```json
[
  {
    "symbol": "TSLY",
    "name": "YieldMax™ TSLA Option Income Strategy ETF",
    "group": "GroupA",
    "frequency": "weekly",
    "metadata": {
      "description": "ETF description",
      "sector": "Technology",
      "exchange": "NYSE Arca",
      "currency": "USD"
    }
  }
]
```

### 3. 배당 스케줄

#### `GET /schedule_v4.json`

배당 스케줄 및 예정된 이벤트를 반환합니다.

**응답 예시:**
```json
{
  "generated_at": "2024-06-12T13:43:01+09:00",
  "events": [
    {
      "symbol": "TSLY",
      "exDate": "2024-06-15T00:00:00Z",
      "payDate": "2024-06-17T00:00:00Z",
      "declareDate": "2024-06-12T00:00:00Z",
      "amount": 0.132,
      "group": "GroupA",
      "frequency": "weekly"
    }
  ],
  "upcoming_events": 79,
  "groups": {
    "GroupA": 11,
    "GroupB": 9,
    "GroupC": 9,
    "GroupD": 9,
    "Weekly": 9,
    "Target12": 5
  }
}
```

**필드 설명:**
- `exDate`: 배당락일 (이 날짜 이후 매수 시 배당 받지 못함)
- `payDate`: 배당 지급일
- `declareDate`: 배당 선언일
- `amount`: 배당금 (달러)

### 4. 개별 ETF 배당 히스토리

#### `GET /dividends_{SYMBOL}.json`

특정 ETF의 배당 히스토리를 반환합니다.

**예시:** `/dividends_TSLY.json`

**응답 예시:**
```json
{
  "symbol": "TSLY",
  "name": "YieldMax™ TSLA Option Income Strategy ETF",
  "group": "GroupA",
  "frequency": "weekly",
  "events": [
    {
      "symbol": "TSLY",
      "exDate": "2024-06-10T00:00:00Z",
      "payDate": "2024-06-12T00:00:00Z",
      "declareDate": "2024-06-07T00:00:00Z",
      "amount": 0.132,
      "group": "GroupA",
      "frequency": "weekly"
    }
  ]
}
```

### 5. API 정보

#### `GET /api_summary_v4.json`

API 정보 및 통계를 반환합니다.

**응답 예시:**
```json
{
  "generated_at": "2024-06-12T13:43:01+09:00",
  "version": "4.0",
  "data_sources": ["YieldMax", "Alpha Vantage", "Financial Modeling Prep"],
  "etf_count": 149,
  "schedule": {
    "total_events": 164,
    "upcoming_events": 79
  },
  "groups": {
    "GroupA": 11,
    "GroupB": 9,
    "GroupC": 9,
    "GroupD": 9,
    "Weekly": 9,
    "Target12": 5
  }
}
```

## 사용 예시

### JavaScript (Fetch API)

```javascript
// ETF 목록 가져오기
fetch('https://your-username.github.io/divminder-crawler/etfs.json')
  .then(response => response.json())
  .then(etfs => {
    console.log(`총 ${etfs.length}개의 ETF`);
    etfs.forEach(etf => {
      console.log(`${etf.symbol}: ${etf.name}`);
    });
  });

// 배당 스케줄 가져오기
fetch('https://your-username.github.io/divminder-crawler/schedule_v4.json')
  .then(response => response.json())
  .then(schedule => {
    console.log(`예정된 배당 이벤트: ${schedule.upcoming_events}개`);
  });
```

### Python (requests)

```python
import requests

# ETF 목록 가져오기
response = requests.get('https://your-username.github.io/divminder-crawler/etfs.json')
etfs = response.json()
print(f"총 {len(etfs)}개의 ETF")

# 특정 ETF 배당 히스토리
response = requests.get('https://your-username.github.io/divminder-crawler/dividends_TSLY.json')
history = response.json()
print(f"TSLY 배당 이벤트: {len(history['events'])}개")
```

### cURL

```bash
# ETF 목록
curl https://your-username.github.io/divminder-crawler/etfs.json

# 배당 스케줄
curl https://your-username.github.io/divminder-crawler/schedule_v4.json

# 특정 ETF 히스토리
curl https://your-username.github.io/divminder-crawler/dividends_TSLY.json
```

## 데이터 업데이트

- **빈도**: 매일 오전 12:05 (KST)
- **소스**: YieldMax 공식 웹사이트, Alpha Vantage API, Financial Modeling Prep API
- **지연시간**: 실시간 데이터가 아님, 최대 24시간 지연 가능

## 에러 처리

### HTTP 상태 코드

- `200 OK`: 성공
- `404 Not Found`: 파일이 존재하지 않음
- `500 Internal Server Error`: 서버 오류

### 에러 응답

API는 표준 HTTP 상태 코드를 사용합니다. JSON 에러 응답은 제공하지 않습니다.

## 제한사항

1. **실시간 데이터 아님**: 최대 24시간 지연
2. **히스토리 제한**: 일부 ETF만 상세 히스토리 제공
3. **API 키 없음**: 인증 기능 없음
4. **Rate Limiting**: GitHub Pages 기본 제한

## 지원

- **GitHub Issues**: [프로젝트 이슈 페이지](https://github.com/your-username/divminder-crawler/issues)
- **문서**: [README.md](../README.md)

## 라이선스

이 API는 MIT 라이선스 하에 제공됩니다. 