#!/usr/bin/env python3
"""
PDF parser for Sberbank statements using py-pdf-parser.
Extracts text in reading order and outputs as plain text
for the FinHelper Sberbank parser.

Usage:
    python pdf_parse.py <input.pdf> [output.txt]
"""
import sys, json
from py_pdf_parser.loaders import load_file


def extract_text(pdf_path: str) -> str:
    """Extract text from PDF in reading order using py-pdf-parser."""
    doc = load_file(pdf_path)
    pages_text = []

    for page in doc.pages:
        # Get all elements on the page, sorted by position (top→bottom, left→right)
        elements = sorted(page.elements, key=lambda e: (-e.bounding_box.y1, e.bounding_box.x0))
        page_lines = [el.text() for el in elements]
        pages_text.append('\n'.join(page_lines))

    return '\n\n'.join(pages_text)


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print('Usage: python pdf_parse.py <input.pdf>', file=sys.stderr)
        sys.exit(1)

    pdf_path = sys.argv[1]
    output_path = sys.argv[2] if len(sys.argv) > 2 else None

    text = extract_text(pdf_path)

    if output_path:
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(text)
        print(f'Extracted {len(text)} chars to {output_path}', file=sys.stderr)
    else:
        print(text)
