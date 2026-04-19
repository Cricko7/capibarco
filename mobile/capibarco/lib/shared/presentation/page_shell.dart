import 'package:flutter/material.dart';

class PageShell extends StatelessWidget {
  const PageShell({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: <Color>[
            colorScheme.surface,
            colorScheme.surfaceContainerLowest,
            colorScheme.secondaryContainer.withValues(alpha: 0.35),
          ],
        ),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) => Stack(
          children: <Widget>[
            Positioned(
              top: -40,
              right: -10,
              child: _GlowCircle(
                color: colorScheme.secondaryContainer.withValues(alpha: 0.45),
                size: 180,
              ),
            ),
            Positioned(
              bottom: -50,
              left: -20,
              child: _GlowCircle(
                color: colorScheme.primaryContainer.withValues(alpha: 0.6),
                size: 220,
              ),
            ),
            SafeArea(
              child: Center(
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    maxWidth: 860,
                    minHeight: constraints.maxHeight,
                  ),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 20,
                      vertical: 16,
                    ),
                    child: child,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _GlowCircle extends StatelessWidget {
  const _GlowCircle({required this.color, required this.size});

  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}
