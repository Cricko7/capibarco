import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/localization/app_localizations.dart';
import 'billing_controller.dart';

class DonateAnimalSheet extends ConsumerStatefulWidget {
  const DonateAnimalSheet({
    required this.animalId,
    required this.animalName,
    required this.ownerDisplayName,
    super.key,
  });

  final String animalId;
  final String animalName;
  final String ownerDisplayName;

  @override
  ConsumerState<DonateAnimalSheet> createState() => _DonateAnimalSheetState();
}

class _DonateAnimalSheetState extends ConsumerState<DonateAnimalSheet> {
  late final TextEditingController _amountController;
  int _selectedAmount = 500;

  @override
  void initState() {
    super.initState();
    _amountController = TextEditingController(text: '500');
    Future<void>.microtask(
      () => ref.read(billingControllerProvider.notifier).reset(),
    );
  }

  @override
  void dispose() {
    _amountController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(billingControllerProvider);

    return Padding(
      padding: EdgeInsets.fromLTRB(
        20,
        24,
        20,
        MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: SafeArea(
        top: false,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                l10n.supportAnimal,
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.w900,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                '${widget.animalName} · ${widget.ownerDisplayName}',
                style: Theme.of(context).textTheme.bodyLarge,
              ),
              const SizedBox(height: 18),
              Text(
                l10n.donationAmount,
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 10),
              Wrap(
                spacing: 10,
                runSpacing: 10,
                children: <int>[300, 500, 1000, 1500]
                    .map(
                      (amount) => ChoiceChip(
                        label: Text('$amount RUB'),
                        selected: _selectedAmount == amount,
                        onSelected: (_) {
                          setState(() {
                            _selectedAmount = amount;
                            _amountController.text = amount.toString();
                          });
                        },
                      ),
                    )
                    .toList(),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _amountController,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  labelText: l10n.donationAmount,
                  hintText: l10n.amountHint,
                ),
              ),
              if (state.errorMessage != null) ...<Widget>[
                const SizedBox(height: 14),
                Text(
                  state.errorMessage!,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
              if (state.intent != null) ...<Widget>[
                const SizedBox(height: 18),
                DecoratedBox(
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primaryContainer,
                    borderRadius: BorderRadius.circular(24),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          l10n.donationIntentCreated,
                          style: Theme.of(context).textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.w900),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          '${state.intent!.amountLabel} · ${state.intent!.status}',
                        ),
                        const SizedBox(height: 12),
                        Text(
                          l10n.paymentUrl,
                          style: const TextStyle(fontWeight: FontWeight.w800),
                        ),
                        const SizedBox(height: 4),
                        SelectableText(state.intent!.paymentUrl),
                        const SizedBox(height: 12),
                        Text(
                          l10n.clientSecret,
                          style: const TextStyle(fontWeight: FontWeight.w800),
                        ),
                        const SizedBox(height: 4),
                        SelectableText(state.intent!.clientSecret),
                      ],
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 20),
              Row(
                children: <Widget>[
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => Navigator.of(context).pop(),
                      child: Text(l10n.close),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: state.isSubmitting
                          ? null
                          : () => _submitDonation(context),
                      icon: state.isSubmitting
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(
                                strokeWidth: 2.2,
                              ),
                            )
                          : const Icon(Icons.volunteer_activism_rounded),
                      label: Text(l10n.donateNow),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _submitDonation(BuildContext context) async {
    final amount = int.tryParse(_amountController.text.trim());
    await ref
        .read(billingControllerProvider.notifier)
        .createAnimalDonation(
          animalId: widget.animalId,
          amountRubles: amount ?? 0,
        );
  }
}
