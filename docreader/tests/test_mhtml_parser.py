import base64
import email
import json
import unittest
from email.header import Header
from email.message import EmailMessage
from pathlib import Path

from docreader.parser.mhtml_parser import MHTMLParser, _header_str
from docreader.parser.registry import registry

REPO_ROOT = Path(__file__).resolve().parents[2]


def _minimal_mhtml_bytes() -> bytes:
    root = EmailMessage()
    root["Subject"] = "Tiny MHTML"
    root.make_related()

    main = EmailMessage()
    main.set_content(
        "<html><body><h1>Main Article</h1>"
        "<p>Hello MHTML world.</p>"
        '<p><a href="chapter03.xhtml#sec2">Chapter 3</a> '
        '<a href="#footnote1">note</a> '
        '<a href="https://example.com">the site</a></p>'
        '<img alt="tiny" src="cid:tiny-image">'
        "<script>window.noise = true</script>"
        "</body></html>",
        subtype="html",
    )
    main["Content-Location"] = "https://example.com/article"
    root.attach(main)

    ad = EmailMessage()
    ad.set_content(
        "<html><body><h1>Advertisement</h1>"
        "<p>Buy this unrelated thing.</p></body></html>",
        subtype="html",
    )
    ad["Content-Location"] = "https://googleads.example/frame.html"
    root.attach(ad)

    image = EmailMessage()
    image.set_content(
        b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR",
        maintype="image",
        subtype="png",
    )
    image["Content-Location"] = "cid:tiny-image"
    root.attach(image)

    return root.as_bytes()


def _mhtml_with_table_image_and_caption() -> bytes:
    root = EmailMessage()
    root["Subject"] = "MHTML with table image"
    root.make_related()

    main = EmailMessage()
    main.set_content(
        "<html><body>"
        "<table>"
        "<tr><th>Leave Type</th><th>Entitlement</th></tr>"
        "<tr><td>Earned Leave</td><td>18 days</td></tr>"
        "</table>"
        '<img src="cid:test-image" alt="चित्र" title="चित्र">'
        "<p>Carry forward is capped at 30 days.</p>"
        "</body></html>",
        subtype="html",
    )
    main["Content-Location"] = "https://example.com/article"
    root.attach(main)

    image = EmailMessage()
    image.set_content(
        b"GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff,\x00\x00"
        b"\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;",
        maintype="image",
        subtype="gif",
    )
    image["Content-ID"] = "<test-image>"
    root.attach(image)

    return root.as_bytes()


def _mhtml_with_8bit_utf8_headers() -> bytes:
    """Browser-style MHTML: raw UTF-8 bytes in MIME headers, not RFC 2047."""
    html_loc = "https://example.com/परीक्षण-पृष्ठ.html".encode("utf-8")
    ad_loc = "https://googleads.example/विज्ञापन.html".encode("utf-8")
    image_loc = "https://example.com/चित्र.png".encode("utf-8")
    content_id = "<चित्र@local>".encode("utf-8")
    attachment_id = "चित्र".encode("utf-8")
    main_html = (
        "<html><body><h1>मुख्य लेख</h1>"
        '<img alt="निरपेक्ष" src="https://example.com/चित्र.png">'
        '<img alt="सापेक्ष" src="चित्र.png">'
        '<img alt="cid" src="cid:चित्र@local">'
        "</body></html>"
    ).encode("utf-8")
    ad_html = "<html><body><h1>विज्ञापन पाठ</h1></body></html>".encode("utf-8")
    png = b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
    return b"".join(
        [
            b"MIME-Version: 1.0\r\n",
            b'Content-Type: multipart/related; boundary="----=_Next"\r\n',
            b"\r\n",
            b"------=_Next\r\n",
            b'Content-Type: text/html; charset="utf-8"\r\n',
            b"Content-Transfer-Encoding: 8bit\r\n",
            b"Content-Location: ",
            html_loc,
            b"\r\n\r\n",
            main_html,
            b"\r\n",
            b"------=_Next\r\n",
            b'Content-Type: text/html; charset="utf-8"\r\n',
            b"Content-Transfer-Encoding: 8bit\r\n",
            b"Content-Location: ",
            ad_loc,
            b"\r\n\r\n",
            ad_html,
            b"\r\n",
            b"------=_Next\r\n",
            b"Content-Type: image/png\r\n",
            b"Content-Transfer-Encoding: base64\r\n",
            b"Content-Location: ",
            image_loc,
            b"\r\nContent-ID: ",
            content_id,
            b"\r\nX-Attachment-Id: ",
            attachment_id,
            b"\r\n\r\n",
            base64.b64encode(png),
            b"\r\n",
            b"------=_Next--\r\n",
        ]
    )


class MHTMLParserTest(unittest.TestCase):
    def test_parse_selects_main_html_and_filters_noise(self):
        document = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        ).parse_into_text(_minimal_mhtml_bytes())

        self.assertIn("Main Article", document.content)
        self.assertIn("Hello MHTML world", document.content)
        self.assertNotIn("Advertisement", document.content)
        self.assertNotIn("window.noise", document.content)
        self.assertEqual(document.metadata["source_format"], "mhtml")

    def test_internal_links_are_unwrapped_but_external_links_remain(self):
        document = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        ).parse_into_text(_minimal_mhtml_bytes())

        self.assertIn("Chapter 3", document.content)
        self.assertIn("note", document.content)
        self.assertNotIn("chapter03.xhtml#sec2", document.content)
        self.assertNotIn("#footnote1", document.content)
        self.assertIn("[the site](https://example.com)", document.content)

    def test_image_extraction_toggle(self):
        with_images = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml", extract_images=True
        ).parse_into_text(_minimal_mhtml_bytes())
        without_images = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml", extract_images=False
        ).parse_into_text(_minimal_mhtml_bytes())

        self.assertEqual(len(with_images.images), 1)
        image_ref = next(iter(with_images.images))
        self.assertTrue(image_ref.startswith("images/"))
        self.assertIn(image_ref, with_images.content)
        self.assertNotIn("cid:tiny-image", with_images.content)
        self.assertEqual(without_images.images, {})

    def test_table_image_and_caption_keep_markdown_block_boundaries(self):
        document = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        ).parse_into_text(_mhtml_with_table_image_and_caption())

        self.assertEqual(len(document.images), 1)
        image_ref = next(iter(document.images))
        self.assertIn(f'![चित्र]({image_ref} "चित्र")', document.content)
        self.assertIn(
            f'| Earned Leave | 18 days |\n\n![चित्र]({image_ref} "चित्र")',
            document.content,
        )
        self.assertIn(
            f'![चित्र]({image_ref} "चित्र")\n\nCarry forward is capped at 30 days.',
            document.content,
        )

    def test_html_to_markdown_preserves_indentation_and_code_block_blanks(self):
        markdown = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        )._html_to_markdown(
            "<ul><li>parent<ul><li>child</li></ul></li></ul>"
            "<blockquote><p>quoted</p></blockquote>"
            "<pre><code>line1\n\n  indented\n</code></pre>"
        )

        self.assertIn("* parent\n  + child", markdown)
        self.assertIn("\n\n> quoted\n\n", markdown)
        self.assertIn("```\nline1\n\n  indented\n```", markdown)

    def test_html_to_markdown_preserves_nested_list_indentation(self):
        markdown = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        )._html_to_markdown("<ul><li>parent<ul><li>child</li></ul></li></ul>")

        self.assertIn("* parent\n  + child", markdown)

    def test_html_to_markdown_preserves_blockquote_boundaries(self):
        markdown = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        )._html_to_markdown("<p>before</p><blockquote><p>quoted</p></blockquote><p>after</p>")

        self.assertIn("before\n\n> quoted\n\nafter", markdown)

    def test_html_to_markdown_preserves_fenced_code_blank_lines(self):
        markdown = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        )._html_to_markdown("<pre><code>line1\n\n\nline2\n</code></pre>")

        self.assertIn("```\nline1\n\n\nline2\n```", markdown)

    def test_html_to_markdown_collapses_excess_blank_lines_outside_code(self):
        markdown = MHTMLParser._normalize_markdown("alpha\n\n  \n\t\n\nbeta")

        self.assertEqual(markdown, "alpha\n\nbeta")
        self.assertNotIn("\n\n\n", markdown)

    def test_html_to_markdown_preserves_hard_break_spaces(self):
        markdown = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        )._html_to_markdown("<p>alpha<br>beta</p>")

        self.assertEqual(markdown, "alpha  \nbeta")

    def test_normalize_markdown_preserves_two_space_hard_break(self):
        markdown = MHTMLParser._normalize_markdown("alpha  \nbeta")

        self.assertEqual(markdown, "alpha  \nbeta")

    def test_html_to_markdown_normalizes_crlf(self):
        markdown = MHTMLParser._normalize_markdown("alpha\r\n\r\nbeta\rgamma")

        self.assertEqual(markdown, "alpha\n\nbeta\ngamma")

    def test_html_to_markdown_does_not_strip_leading_indentation_at_document_start(self):
        markdown = MHTMLParser._normalize_markdown("  indented start\n")

        self.assertEqual(markdown, "  indented start")

    def test_mhtml_shared_contract_fixture(self):
        fixture = REPO_ROOT / "testdata" / "mhtml" / "titled-image.mhtml"
        contract_path = REPO_ROOT / "testdata" / "mhtml" / "titled-image-contract.json"
        contract = json.loads(contract_path.read_text(encoding="utf-8"))

        document = MHTMLParser(
            file_name="titled-image.mhtml", file_type="mhtml"
        ).parse_into_text(fixture.read_bytes())

        self.assertEqual(document.content, contract["markdown_content"])
        self.assertEqual(len(document.images), 1)
        image_contract = contract["images"][0]
        self.assertIn(image_contract["original_ref"], document.images)
        self.assertEqual(
            base64.b64decode(document.images[image_contract["original_ref"]]),
            base64.b64decode(image_contract["image_data_base64"]),
        )
        self.assertIn(
            '| Earned Leave | 18 days |\n\n![चित्र](images/पृष्ठ 1 (परीक्षण).gif "चरण 1) चित्र")',
            document.content,
        )
        self.assertIn(
            '![चित्र](images/पृष्ठ 1 (परीक्षण).gif "चरण 1) चित्र")\n\nCarry forward is capped at 30 days.',
            document.content,
        )
        self.assertNotIn("\n\n\n", document.content.split("```", 1)[0])
        self.assertIn("```\nline1\n\n\nline2\n```", document.content)
        self.assertIn("* parent\n  + child", document.content)
        self.assertIn("alpha  \nbeta", document.content)

    def test_header_str_recovers_unknown_8bit_utf8(self):
        raw = (
            b"Content-Type: text/plain\r\n"
            b"Content-Location: https://example.com/"
            + "चित्र.png".encode("utf-8")
            + b"\r\n\r\nbody\r\n"
        )
        value = email.message_from_bytes(raw).get("Content-Location")
        self.assertIsInstance(value, Header)
        self.assertEqual(_header_str(value), "https://example.com/चित्र.png")
        self.assertEqual(_header_str(None), "")
        self.assertEqual(_header_str("ascii-location"), "ascii-location")
        self.assertEqual(_header_str("पहलेसेunicode"), "पहलेसेunicode")

    def test_parse_8bit_utf8_headers_rewrites_images_without_crashing(self):
        document = MHTMLParser(
            file_name="article.mhtml", file_type="mhtml"
        ).parse_into_text(_mhtml_with_8bit_utf8_headers())

        self.assertIn("मुख्य लेख", document.content)
        self.assertNotIn("विज्ञापन पाठ", document.content)
        self.assertIn("images/चित्र.png", document.images)
        self.assertIn("images/चित्र.png", document.content)
        self.assertEqual(document.content.count("images/चित्र.png"), 3)
        self.assertNotIn("cid:चित्र@local", document.content)
        self.assertNotIn("\ufffd", document.content)
        self.assertTrue(all("\ufffd" not in key for key in document.images))

    def test_registry_resolves_mhtml(self):
        self.assertIs(registry.get_parser_class("", "mhtml"), MHTMLParser)


if __name__ == "__main__":
    unittest.main()
