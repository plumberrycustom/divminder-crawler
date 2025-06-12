# DivMinder Crawler

YieldMax ETF 배당 데이터를 수집하고 JSON API를 생성하는 Go 크롤러입니다.

## 기능

- YieldMax 공식 웹사이트에서 ETF 스케줄 데이터 수집
- Alpha Vantage API를 통한 ETF 메타데이터 보강
- Financial Modeling Prep API를 통한 배당 히스토리 수집
- GitHub Pages용 정적 JSON API 생성
- 자동화된 일일 데이터 업데이트

## 수집 데이터

### ETF 목록 (`etfs.json`)
- YieldMax Target 12 ETF (Group A-D)
- YieldMax Weekly ETF
- YieldMax Monthly ETF

### 배당 스케줄 (`schedule.json`)
- 향후 배당 이벤트
- Ex-Date, Pay-Date, Declare-Date
- 그룹별 분류

### 개별 ETF 히스토리 (`dividends_{SYMBOL}.json`)
- 과거 배당 히스토리
- 배당금 변화 추이
- 통계 정보

## 사용법

### 수동 실행
```bash
go run cmd/crawler/main.go
```

### 환경 변수
```bash
ALPHA_VANTAGE_API_KEY=your_api_key
FMP_API_KEY=your_api_key  # 선택사항
```

## 배포

GitHub Actions를 통해 매일 00:05 KST에 자동 실행됩니다.

## 개발

```bash
# 의존성 설치
go mod tidy

# 테스트 실행
go test ./...

# 빌드
go build -o crawler cmd/crawler/main.go
``` 