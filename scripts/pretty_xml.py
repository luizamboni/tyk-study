#!/usr/bin/env python3
import sys, xml.dom.minidom
print(xml.dom.minidom.parseString(sys.stdin.buffer.read()).toprettyxml())
