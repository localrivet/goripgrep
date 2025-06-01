#!/usr/bin/env python3

lines = [
    'github.com/BurntSushi/locker,v0.0.0-20171006230638-a6e239ea1c69,h1:+tu3HOoMXB7RXEINRVIpxJCT+KdYiI7LAEAUrOw3dIU=,836038343df9e9126b59d54201951191898bd875ec32d93c2018d759f358fcfb\n',
    'github.com/BurntSushi/toml,v0.3.1,h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=,815c6e594745f2d8842ff9a4b0569c6695e6cdfd5e07e5b3d98d06b72ca41e3c\n',
    'github.com/BurntSushi/xgb,v0.0.0-20160522181843-27f122750802,h1:1BDTz0u9nC3//pOCMdNH+CiXJVYJh5UQNCOBG7jbELc=,f52962c7fbeca81ea8a777d1f8b1f1d25803dc437fbb490f253344232884328e\n',
    'github.com/BurntSushi/xgbutil,v0.0.0-20190907113008-ad855c713046,h1:O/r2Sj+8QcMF7V5IcmiE2sMFV2q3J47BEirxbXJAdzA=,492ce6b11d7faaec4e15d1279d81e28d2e0e9844ad117f9de9411286a5b0e305\n',
    'github.com/other/package,v1.0.0,h1:SomeHashHere,somehashhere\n' * 1000,
]

with open('large_test.csv', 'w') as f:
    for line in lines * 500:  # Create a large file
        f.write(line)

print('Created large_test.csv') 