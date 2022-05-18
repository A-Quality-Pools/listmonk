package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_1_0 performs the DB migrations for v.2.1.0.
func V2_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	//Add MJML types
	if _, err := db.Exec(`
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'template_type') THEN
			CREATE TYPE template_type AS ENUM ('html', 'mjml');
		END IF;
	END$$;

	ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'mjml';

	ALTER TABLE templates ADD COLUMN IF NOT EXISTS type template_type NOT NULL DEFAULT 'html';
	`); err != nil {
		return err
	}

	// Insert appearance related settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('appearance.admin.custom_css', '""'),
 			('appearance.admin.custom_js', '""'),
 			('appearance.public.custom_css', '""'),
 			('appearance.public.custom_js', '""'),
 			('upload.s3.public_url', '""')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	// Replace all `tls_enabled: true/false` keys in the `smtp` settings JSON array
	// with the new field `tls_type: STARTTLS|TLS|none`.
	// The `tls_enabled` key is removed.
	if _, err := db.Exec(`
		UPDATE settings SET value = s.updated
		FROM (
			SELECT JSONB_AGG(
				JSONB_SET(v - 'tls_enabled', '{tls_type}', (CASE WHEN v->>'tls_enabled' = 'true' THEN '"STARTTLS"' ELSE '"none"' END)::JSONB)
			) AS updated FROM settings, JSONB_ARRAY_ELEMENTS(value) v WHERE key = 'smtp'
		) s WHERE key = 'smtp' AND value::TEXT LIKE '%tls_enabled%';
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS headers JSONB NOT NULL DEFAULT '[]';`); err != nil {
		return err
	}

	return nil
}
