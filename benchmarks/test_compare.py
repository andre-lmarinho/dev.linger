import unittest

from compare import parse


class ParseTest(unittest.TestCase):
    def test_parses_integer_and_fractional_timings(self):
        output = (
            "BenchmarkLayout-12 1000 123 ns/op 14761 B/op 235 allocs/op\n"
            "BenchmarkFast-12 1000 9.87 ns/op 12 B/op 1 allocs/op"
        )
        self.assertEqual(
            parse(output),
            {
                "BenchmarkLayout": (123.0, 14761, 235),
                "BenchmarkFast": (9.87, 12, 1),
            },
        )
