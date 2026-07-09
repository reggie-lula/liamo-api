# Seedance 2 latest pricing update

## Scope

Update the Doubao task Seedance 2 price tables to match the latest confirmed
pricing:

| Model | 480p/720p with video input | 480p/720p without video input | 1080p with video input | 1080p without video input | 4K with video input | 4K without video input |
|-------|----------------------------|-------------------------------|------------------------|---------------------------|---------------------|------------------------|
| `doubao-seedance-2-0-260128` | ¥28 / 1M tokens | ¥46 / 1M tokens | ¥31 / 1M tokens | ¥51 / 1M tokens | ¥16 / 1M tokens | ¥26 / 1M tokens |
| `doubao-seedance-2-0-fast-260128` | ¥22 / 1M tokens | ¥37 / 1M tokens | - | - | - | - |
| `doubao-seedance-2-0-mini-260615` | ¥14 / 1M tokens | ¥23 / 1M tokens | - | - | - | - |

## Implementation Notes

- `doubao-seedance-2-0-mini-260615` is added to the Doubao task model list.
- `seedanceTierPriceMap` now includes Mini low-tier pricing.
- Fast keeps the confirmed low-tier no-video price at `37`.
- Fast and Mini do not declare 1080p or 4K prices. If a missing tier is
  requested, the estimator falls back to the same model's low-tier price
  instead of borrowing another model's higher-resolution price.
- Pro keeps the full 480p/720p, 1080p, and 4K matrix.
