package mirror

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kumbuka-me/kumbuka/pkg/domain"
)

// PostgresRepository provides the read-only PostgreSQL queries required by mirror export.
type PostgresRepository struct {
	// pool owns the PostgreSQL connections used by mirror queries.
	pool *pgxpool.Pool
}

// OpenPostgres connects to an existing Kumbuka PostgreSQL database for read-only mirror queries.
func OpenPostgres(ctx context.Context, url string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

// Close releases the PostgreSQL connection pool.
func (r *PostgresRepository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

// PageInventory returns active pages in canonical path order.
func (r *PostgresRepository) PageInventory(ctx context.Context) ([]domain.Page, error) {
	rows, err := r.pool.Query(ctx, `
SELECT p.id,p.slug,p.title,p.status,coalesce(p.owner_group_id,0),coalesce(g.name,''),p.last_reviewed_at,p.review_interval_days,p.updated_at
FROM pages p
LEFT JOIN wiki_groups g ON g.id=p.owner_group_id
WHERE p.deleted_at IS NULL
ORDER BY p.slug`)
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()

	var pages []domain.Page
	for rows.Next() {
		var page domain.Page
		if err := rows.Scan(
			&page.ID,
			&page.Slug,
			&page.Title,
			&page.Status,
			&page.OwnerGroupID,
			&page.OwnerGroup,
			&page.LastReviewedAt,
			&page.ReviewIntervalDays,
			&page.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan page inventory: %w", err)
		}
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}

	return pages, nil
}

// GetPage returns complete mirror metadata and Markdown for one active page.
func (r *PostgresRepository) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	var page domain.Page
	err := r.pool.QueryRow(ctx, `
SELECT
  p.id,
  p.slug,
  p.title,
  coalesce(ni.icon,''),
  p.markdown_content,
  coalesce(p.created_by,0),
  coalesce(p.updated_by,0),
  coalesce(u.display_name,u.username,''),
  p.created_at,
  p.updated_at,
  p.view_count,
  p.status,
  p.content_language,
  coalesce(p.owner_group_id,0),
  coalesce(g.name,''),
  p.last_reviewed_at,
  p.review_interval_days,
  p.deprecated_target
FROM pages p
LEFT JOIN navigation_icons ni ON ni.path=p.slug
LEFT JOIN users u ON u.id=p.updated_by
LEFT JOIN wiki_groups g ON g.id=p.owner_group_id
WHERE p.slug=$1 AND p.deleted_at IS NULL`, slug).Scan(
		&page.ID,
		&page.Slug,
		&page.Title,
		&page.Icon,
		&page.Markdown,
		&page.CreatedBy,
		&page.UpdatedBy,
		&page.Author,
		&page.CreatedAt,
		&page.UpdatedAt,
		&page.ViewCount,
		&page.Status,
		&page.Language,
		&page.OwnerGroupID,
		&page.OwnerGroup,
		&page.LastReviewedAt,
		&page.ReviewIntervalDays,
		&page.DeprecatedTarget,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Page{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Page{}, fmt.Errorf("read page %q: %w", slug, err)
	}

	page.Tags, err = r.pageTags(ctx, page.ID)
	if err != nil {
		return domain.Page{}, err
	}
	page.Groups, err = r.pageGroups(ctx, page.ID)
	if err != nil {
		return domain.Page{}, err
	}
	page.Properties, err = r.pageProperties(ctx, page.ID)
	if err != nil {
		return domain.Page{}, err
	}

	return page, nil
}

// Images returns the stored image identifiers required by mirror export.
func (r *PostgresRepository) Images(ctx context.Context) ([]domain.Image, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id,filename
FROM images
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	defer rows.Close()

	var images []domain.Image
	for rows.Next() {
		var image domain.Image
		if err := rows.Scan(&image.ID, &image.Filename); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, image)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}

	return images, nil
}

// ImageContent returns the binary payload for one stored image.
func (r *PostgresRepository) ImageContent(ctx context.Context, id int64) (domain.ImageData, error) {
	var image domain.ImageData
	err := r.pool.QueryRow(ctx, `
SELECT filename,content_type,data
FROM images
WHERE id=$1`, id).Scan(&image.Filename, &image.ContentType, &image.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ImageData{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ImageData{}, fmt.Errorf("read image %d: %w", id, err)
	}

	return image, nil
}

// Attachments returns the stored attachment identifiers required by mirror export.
func (r *PostgresRepository) Attachments(ctx context.Context) ([]domain.Attachment, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id,filename
FROM attachments
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []domain.Attachment
	for rows.Next() {
		var attachment domain.Attachment
		if err := rows.Scan(&attachment.ID, &attachment.Filename); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}

	return attachments, nil
}

// AttachmentContent returns metadata and bytes for one stored attachment.
func (r *PostgresRepository) AttachmentContent(ctx context.Context, id int64) (domain.AttachmentData, error) {
	var attachment domain.AttachmentData
	err := r.pool.QueryRow(ctx, `
SELECT id,filename,content_type,size_bytes,coalesce(uploaded_by,0),created_at,data
FROM attachments
WHERE id=$1`, id).Scan(
		&attachment.ID,
		&attachment.Filename,
		&attachment.ContentType,
		&attachment.SizeBytes,
		&attachment.UploadedBy,
		&attachment.CreatedAt,
		&attachment.Data,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AttachmentData{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.AttachmentData{}, fmt.Errorf("read attachment %d: %w", id, err)
	}

	return attachment, nil
}

// pageTags returns normalized tags assigned to one page.
func (r *PostgresRepository) pageTags(ctx context.Context, pageID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
SELECT t.name
FROM tags t
JOIN page_tags pt ON pt.tag_id=t.id
WHERE pt.page_id=$1
ORDER BY t.name`, pageID)
	if err != nil {
		return nil, fmt.Errorf("list page tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan page tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list page tags: %w", err)
	}

	return tags, nil
}

// pageGroups returns collaboration groups assigned to one page.
func (r *PostgresRepository) pageGroups(ctx context.Context, pageID int64) ([]domain.Group, error) {
	rows, err := r.pool.Query(ctx, `
SELECT g.id,g.name
FROM wiki_groups g
JOIN page_groups pg ON pg.group_id=g.id
WHERE pg.page_id=$1
ORDER BY lower(g.name),g.id`, pageID)
	if err != nil {
		return nil, fmt.Errorf("list page groups: %w", err)
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var group domain.Group
		if err := rows.Scan(&group.ID, &group.Name); err != nil {
			return nil, fmt.Errorf("scan page group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list page groups: %w", err)
	}

	return groups, nil
}

// pageProperties returns structured metadata assigned to one page.
func (r *PostgresRepository) pageProperties(ctx context.Context, pageID int64) ([]domain.PageProperty, error) {
	rows, err := r.pool.Query(ctx, `
SELECT key,value
FROM page_properties
WHERE page_id=$1
ORDER BY lower(key),key`, pageID)
	if err != nil {
		return nil, fmt.Errorf("list page properties: %w", err)
	}
	defer rows.Close()

	var properties []domain.PageProperty
	for rows.Next() {
		var property domain.PageProperty
		if err := rows.Scan(&property.Key, &property.Value); err != nil {
			return nil, fmt.Errorf("scan page property: %w", err)
		}
		properties = append(properties, property)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list page properties: %w", err)
	}

	return properties, nil
}

var _ repository = (*PostgresRepository)(nil)
