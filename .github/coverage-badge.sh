#!/bin/sh
# Writes the coverage badge to stdout as SVG. $1 is the total coverage as
# go tool cover prints it, e.g. 87.3%. The badge has one color: it shows the
# value and sets no threshold.
set -eu

value=$1
if ! printf '%s\n' "$value" | grep -Eqx '[0-9]{1,3}(\.[0-9]+)?%'; then
	echo "coverage-badge: not a percentage: $value" >&2
	exit 1
fi

label=coverage
label_w=61
value_w=$(( ${#value} * 13 / 2 + 12 ))
total_w=$(( label_w + value_w ))
# Text is drawn at 10x scale, as in the shields.io flat style.
label_x=$(( label_w * 5 ))
value_x=$(( (label_w * 2 + value_w) * 5 ))
label_len=$(( (label_w - 10) * 10 ))
value_len=$(( (value_w - 10) * 10 ))

cat <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$total_w" height="20" role="img" aria-label="$label: $value">
<title>$label: $value</title>
<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>
<clipPath id="r"><rect width="$total_w" height="20" rx="3" fill="#fff"/></clipPath>
<g clip-path="url(#r)"><rect width="$label_w" height="20" fill="#555"/><rect x="$label_w" width="$value_w" height="20" fill="#007ec6"/><rect width="$total_w" height="20" fill="url(#s)"/></g>
<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">
<text aria-hidden="true" x="$label_x" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="$label_len">$label</text>
<text x="$label_x" y="140" transform="scale(.1)" textLength="$label_len">$label</text>
<text aria-hidden="true" x="$value_x" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="$value_len">$value</text>
<text x="$value_x" y="140" transform="scale(.1)" textLength="$value_len">$value</text>
</g>
</svg>
EOF
