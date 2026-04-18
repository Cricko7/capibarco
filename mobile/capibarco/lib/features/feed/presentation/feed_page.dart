import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/animal_details_sheet.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/stale_banner.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../billing/presentation/donate_animal_sheet.dart';
import '../../profile/presentation/profile_controller.dart';
import 'feed_controller.dart';

class FeedPage extends ConsumerStatefulWidget {
  const FeedPage({super.key});

  @override
  ConsumerState<FeedPage> createState() => _FeedPageState();
}

class _FeedPageState extends ConsumerState<FeedPage> {
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref.read(feedControllerProvider.notifier).load(),
    );
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      if (!mounted) {
        return;
      }
      ref.read(feedControllerProvider.notifier).refreshRealtime();
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
    final state = ref.watch(feedControllerProvider);
    final currentProfileId =
        ref.watch(profileControllerProvider).profile?.id ??
        ref.watch(authControllerProvider).session?.user.id;

    if (state.isLoading && state.cards.isEmpty) {
      return Scaffold(body: StatusView.loading(message: l10n.loading));
    }

    if (state.errorMessage != null && state.cards.isEmpty) {
      return Scaffold(
        body: StatusView.message(
          message: state.errorMessage!,
          icon: Icons.error_outline_rounded,
          action: FilledButton(
            onPressed: () => ref.read(feedControllerProvider.notifier).load(),
            child: Text(l10n.retry),
          ),
        ),
      );
    }

    return Scaffold(
      body: PageShell(
        child: RefreshIndicator(
          onRefresh: () => ref.read(feedControllerProvider.notifier).load(),
          child: ListView(
            children: <Widget>[
              SectionHeader(
                title: l10n.feed,
                subtitle: 'Swipe-ready feed powered by the PetMatch gateway.',
              ),
              if (state.isStale) ...<Widget>[
                const SizedBox(height: 16),
                StaleBanner(message: l10n.staleData),
              ],
              const SizedBox(height: 16),
              if (state.cards.isEmpty)
                StatusView.message(
                  message: l10n.emptyFeed,
                  icon: Icons.pets_outlined,
                )
              else
                ...state.cards.map((card) {
                  final isOwnCard =
                      currentProfileId != null &&
                      currentProfileId == card.ownerProfileId;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: InkWell(
                      borderRadius: BorderRadius.circular(28),
                      onTap: () => _showAnimalDetails(context, ref, card),
                      child: SoftCard(
                        padding: EdgeInsets.zero,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: <Widget>[
                            ClipRRect(
                              borderRadius: const BorderRadius.vertical(
                                top: Radius.circular(28),
                              ),
                              child: SizedBox(
                                height: 220,
                                width: double.infinity,
                                child: card.photoUrl.isNotEmpty
                                    ? Image.network(
                                        card.photoUrl,
                                        fit: BoxFit.cover,
                                      )
                                    : DecoratedBox(
                                        decoration: BoxDecoration(
                                          gradient: LinearGradient(
                                            colors: <Color>[
                                              Theme.of(
                                                context,
                                              ).colorScheme.primaryContainer,
                                              Theme.of(
                                                context,
                                              ).colorScheme.secondaryContainer,
                                            ],
                                          ),
                                        ),
                                        child: Icon(
                                          Icons.pets_rounded,
                                          size: 52,
                                          color: Theme.of(
                                            context,
                                          ).colorScheme.primary,
                                        ),
                                      ),
                              ),
                            ),
                            Padding(
                              padding: const EdgeInsets.all(20),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: <Widget>[
                                  Row(
                                    children: <Widget>[
                                      Expanded(
                                        child: Text(
                                          '${card.name}, ${card.speciesLabel}',
                                          style: Theme.of(context)
                                              .textTheme
                                              .headlineSmall
                                              ?.copyWith(
                                                fontWeight: FontWeight.w900,
                                              ),
                                        ),
                                      ),
                                      if (card.boosted)
                                        Chip(
                                          avatar: const Icon(
                                            Icons.bolt_rounded,
                                            size: 18,
                                          ),
                                          label: const Text('Boosted'),
                                        ),
                                    ],
                                  ),
                                  const SizedBox(height: 8),
                                  Text(
                                    '${card.ownerDisplayName}${card.city.isNotEmpty ? ' · ${card.city}' : ''}',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.titleMedium,
                                  ),
                                  const SizedBox(height: 12),
                                  Text(
                                    card.description.isEmpty
                                        ? 'No description yet.'
                                        : card.description,
                                  ),
                                  if (card
                                      .rankingReasons
                                      .isNotEmpty) ...<Widget>[
                                    const SizedBox(height: 14),
                                    Wrap(
                                      spacing: 8,
                                      runSpacing: 8,
                                      children: card.rankingReasons
                                          .take(3)
                                          .map(
                                            (reason) =>
                                                Chip(label: Text(reason)),
                                          )
                                          .toList(),
                                    ),
                                  ],
                                  const SizedBox(height: 14),
                                  Align(
                                    alignment: Alignment.centerLeft,
                                    child: TextButton.icon(
                                      onPressed: () =>
                                          showModalBottomSheet<void>(
                                            context: context,
                                            isScrollControlled: true,
                                            backgroundColor: Theme.of(context)
                                                .colorScheme
                                                .surfaceContainerLowest,
                                            shape: const RoundedRectangleBorder(
                                              borderRadius:
                                                  BorderRadius.vertical(
                                                    top: Radius.circular(32),
                                                  ),
                                            ),
                                            builder: (context) =>
                                                DonateAnimalSheet(
                                                  animalId: card.animalId,
                                                  animalName: card.name,
                                                  ownerDisplayName:
                                                      card.ownerDisplayName,
                                                ),
                                          ),
                                      icon: const Icon(
                                        Icons.volunteer_activism_rounded,
                                      ),
                                      label: Text(l10n.supportAnimal),
                                    ),
                                  ),
                                  const SizedBox(height: 18),
                                  Row(
                                    children: <Widget>[
                                      Expanded(
                                        child: OutlinedButton.icon(
                                          onPressed: isOwnCard
                                              ? null
                                              : () => ref
                                                    .read(
                                                      feedControllerProvider
                                                          .notifier,
                                                    )
                                                    .swipe(
                                                      card: card,
                                                      liked: false,
                                                    ),
                                          icon: const Icon(Icons.close_rounded),
                                          label: Text(l10n.pass),
                                        ),
                                      ),
                                      const SizedBox(width: 12),
                                      Expanded(
                                        child: FilledButton.icon(
                                          onPressed: isOwnCard
                                              ? null
                                              : () => ref
                                                    .read(
                                                      feedControllerProvider
                                                          .notifier,
                                                    )
                                                    .swipe(
                                                      card: card,
                                                      liked: true,
                                                    ),
                                          icon: const Icon(
                                            Icons.favorite_rounded,
                                          ),
                                          label: Text(l10n.like),
                                        ),
                                      ),
                                    ],
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                }),
              if (state.nextPageToken.isNotEmpty) ...<Widget>[
                const SizedBox(height: 8),
                FilledButton.tonal(
                  onPressed: state.isLoadingMore
                      ? null
                      : () => ref
                            .read(feedControllerProvider.notifier)
                            .loadMore(),
                  child: state.isLoadingMore
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2.4),
                        )
                      : const Text('Load more'),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _showAnimalDetails(
    BuildContext context,
    WidgetRef ref,
    dynamic card,
  ) {
    return showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (sheetContext) => AnimalDetailsSheet(
        name: card.name,
        subtitle:
            '${card.speciesLabel}${card.city.isNotEmpty ? ' · ${card.city}' : ''}',
        description: card.description,
        photoUrl: card.photoUrl,
        respondLabel: _isOwnCard(ref, card) ? 'Own card' : 'Respond',
        onRespond: _isOwnCard(ref, card)
            ? null
            : () async {
                await ref
                    .read(feedControllerProvider.notifier)
                    .swipe(card: card, liked: true);
                if (sheetContext.mounted) {
                  Navigator.of(sheetContext).pop();
                }
              },
      ),
    );
  }

  bool _isOwnCard(WidgetRef ref, dynamic card) {
    final currentProfileId =
        ref.read(profileControllerProvider).profile?.id ??
        ref.read(authControllerProvider).session?.user.id;
    return currentProfileId != null && currentProfileId == card.ownerProfileId;
  }
}
