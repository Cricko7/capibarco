import 'package:flutter/material.dart';

import 'soft_card.dart';

class StatusView extends StatelessWidget {
  const StatusView.loading({this.message = 'Loading...', super.key})
    : icon = null,
      action = null;

  const StatusView.message({
    required this.message,
    this.icon,
    this.action,
    super.key,
  });

  final String message;
  final IconData? icon;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: SoftCard(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            if (icon != null) ...<Widget>[
              Icon(icon, size: 42),
              const SizedBox(height: 12),
            ] else ...<Widget>[
              const SizedBox(
                width: 28,
                height: 28,
                child: CircularProgressIndicator(strokeWidth: 3),
              ),
              const SizedBox(height: 12),
            ],
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            if (action != null) ...<Widget>[
              const SizedBox(height: 16),
              action!,
            ],
          ],
        ),
      ),
    );
  }
}
