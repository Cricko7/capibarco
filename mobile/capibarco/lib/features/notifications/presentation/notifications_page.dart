import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/stale_banner.dart';
import '../../../shared/presentation/status_view.dart';
import 'notifications_controller.dart';

class NotificationsPage extends ConsumerStatefulWidget {
  const NotificationsPage({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref.read(notificationsControllerProvider.notifier).load(),
    );
    _refreshTimer = Timer.periodic(const Duration(seconds: 6), (_) {
      if (!mounted) {
        return;
      }
      ref.read(notificationsControllerProvider.notifier).refreshRealtime();
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(notificationsControllerProvider);
    final dateFormat = DateFormat.yMMMd(
      Localizations.localeOf(context).languageCode,
    ).add_Hm();

    return Scaffold(
      body: PageShell(
        child: RefreshIndicator(
          onRefresh: () =>
              ref.read(notificationsControllerProvider.notifier).load(),
          child: ListView(
            children: <Widget>[
              SectionHeader(
                title: l10n.notifications,
                subtitle:
                    'Inbox from notification-service and mark-read gateway commands.',
              ),
              if (state.isStale) ...<Widget>[
                const SizedBox(height: 16),
                StaleBanner(message: l10n.staleData),
              ],
              const SizedBox(height: 16),
              if (state.isLoading && state.items.isEmpty)
                StatusView.loading(message: l10n.loading)
              else if (state.errorMessage != null && state.items.isEmpty)
                StatusView.message(
                  message: state.errorMessage!,
                  icon: Icons.error_outline_rounded,
                  action: FilledButton(
                    onPressed: () => ref
                        .read(notificationsControllerProvider.notifier)
                        .load(),
                    child: Text(l10n.retry),
                  ),
                )
              else if (state.items.isEmpty)
                StatusView.message(
                  message: l10n.emptyNotifications,
                  icon: Icons.notifications_none_rounded,
                )
              else
                ...state.items.map(
                  (item) => Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: SoftCard(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Row(
                            children: <Widget>[
                              Expanded(
                                child: Text(
                                  item.title,
                                  style: Theme.of(context).textTheme.titleLarge
                                      ?.copyWith(fontWeight: FontWeight.w800),
                                ),
                              ),
                              if (item.readAt == null)
                                FilledButton.tonal(
                                  onPressed: () => ref
                                      .read(
                                        notificationsControllerProvider
                                            .notifier,
                                      )
                                      .markAsRead(item),
                                  child: const Text('Mark read'),
                                ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Text(item.body),
                          const SizedBox(height: 12),
                          Text(
                            '${item.type} · ${dateFormat.format(item.createdAt.toLocal())}',
                            style: Theme.of(context).textTheme.bodySmall,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
