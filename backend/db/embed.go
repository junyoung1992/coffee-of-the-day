package db

import "embed"

// MigrationsFS는 db/migrations 디렉토리의 SQL 파일을 바이너리에 embed한다.
// 컨테이너 환경에서 실행 위치에 관계없이 마이그레이션을 실행할 수 있게 한다.
//
//go:embed all:migrations
var MigrationsFS embed.FS
