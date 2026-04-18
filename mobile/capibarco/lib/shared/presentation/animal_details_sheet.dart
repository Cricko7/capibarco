import 'package:flutter/material.dart';

class AnimalDetailsSheet extends StatelessWidget {
  const AnimalDetailsSheet({
    required this.name,
    required this.subtitle,
    required this.description,
    required this.photoUrl,
    required this.respondLabel,
    this.onRespond,
    this.statusLabel = '',
    this.isResponding = false,
    super.key,
  });

  final String name;
  final String subtitle;
  final String description;
  final String photoUrl;
  final String statusLabel;
  final String respondLabel;
  final bool isResponding;
  final VoidCallback? onRespond;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Center(
              child: Container(
                width: 44,
                height: 4,
                decoration: BoxDecoration(
                  color: theme.colorScheme.outlineVariant,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
            ),
            const SizedBox(height: 16),
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                height: 240,
                width: double.infinity,
                child: photoUrl.isEmpty
                    ? DecoratedBox(
                        decoration: BoxDecoration(
                          color: theme.colorScheme.primaryContainer,
                        ),
                        child: Icon(
                          Icons.pets_rounded,
                          size: 64,
                          color: theme.colorScheme.primary,
                        ),
                      )
                    : Image.network(photoUrl, fit: BoxFit.cover),
              ),
            ),
            const SizedBox(height: 16),
            Text(
              name.isEmpty ? 'Pet' : name,
              style: theme.textTheme.headlineSmall?.copyWith(
                fontWeight: FontWeight.w900,
              ),
            ),
            if (subtitle.isNotEmpty) ...<Widget>[
              const SizedBox(height: 6),
              Text(subtitle, style: theme.textTheme.titleMedium),
            ],
            if (statusLabel.isNotEmpty) ...<Widget>[
              const SizedBox(height: 12),
              Chip(label: Text(statusLabel)),
            ],
            const SizedBox(height: 14),
            Text(
              description.isEmpty ? 'No description yet.' : description,
              style: theme.textTheme.bodyLarge,
            ),
            const SizedBox(height: 20),
            Row(
              children: <Widget>[
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: () => Navigator.of(context).pop(),
                    icon: const Icon(Icons.close_rounded),
                    label: const Text('Закрыть'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: isResponding ? null : onRespond,
                    icon: isResponding
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.favorite_rounded),
                    label: Text(respondLabel),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
