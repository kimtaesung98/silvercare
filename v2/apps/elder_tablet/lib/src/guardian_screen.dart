import 'package:battery_plus/battery_plus.dart';
import 'package:flutter/material.dart';

import 'chat_client.dart';
import 'tokens.dart';
import 'voice.dart';

/// 충전 여부를 묻는 자리. 시험에서는 가짜로 바꿔 끼웁니다.
abstract class Charger {
  Future<bool> plugged();
}

class DeviceCharger implements Charger {
  final _battery = Battery();

  @override
  Future<bool> plugged() async {
    try {
      final s = await _battery.batteryState;
      return s == BatteryState.charging || s == BatteryState.full || s == BatteryState.connectedNotCharging;
    } catch (_) {
      return false;
    }
  }
}

/// 보호자 "말동무 켜 두기" 화면 (피그마 11:71). 충전·인터넷·마이크를 확인하고,
/// 어르신 이름을 적은 뒤 "켜 두고 나가기"를 누르면 어르신 화면으로 넘어갑니다.
class GuardianScreen extends StatefulWidget {
  const GuardianScreen({
    super.key,
    required this.chat,
    required this.voice,
    required this.charger,
    required this.onKeepOn,
    this.initialName = '',
  });

  final Chat chat;
  final Voice voice;
  final Charger charger;
  final String initialName;
  final void Function(String elderName) onKeepOn;

  @override
  State<GuardianScreen> createState() => _GuardianScreenState();
}

class _GuardianScreenState extends State<GuardianScreen> {
  bool? _charging, _online, _mic;
  late final _name = TextEditingController(text: widget.initialName);

  @override
  void initState() {
    super.initState();
    _name.addListener(() => setState(() {}));
    _check();
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _check() async {
    setState(() => _charging = _online = _mic = null);
    final results = await Future.wait([widget.charger.plugged(), widget.chat.healthy(), widget.voice.init()]);
    if (!mounted) return;
    setState(() {
      _charging = results[0];
      _online = results[1];
      _mic = results[2];
    });
  }

  bool get _ready => _online == true && _mic == true && _name.text.trim().isNotEmpty;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Tokens.background,
      body: FigmaFrame(
        child: Stack(
          children: [
            const Positioned(left: 72, top: 230, width: 520, child: _Intro()),
            Positioned(
              left: 648,
              top: 142,
              width: 560,
              child: Container(
                padding: const EdgeInsets.fromLTRB(36, 28, 36, 32),
                decoration: BoxDecoration(color: Tokens.white, borderRadius: BorderRadius.circular(32)),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _CheckRow(_charging, '충전기에 꽂혀 있어요', '충전기를 꽂아 주세요'),
                    _CheckRow(_online, '인터넷이 연결됐어요', '서버에 연결되지 않아요'),
                    _CheckRow(
                      _mic,
                      '마이크가 잘 들려요',
                      '마이크를 허용해 주세요',
                      trailing: TextButton(
                        onPressed: _check,
                        child: const Text('다시 확인', style: TextStyle(fontSize: 26, color: Tokens.accent)),
                      ),
                    ),
                    _Row(
                      '어르신',
                      SizedBox(
                        width: 240,
                        child: TextField(
                          key: const Key('elder-name'),
                          controller: _name,
                          textAlign: TextAlign.right,
                          style: _value,
                          decoration: const InputDecoration(
                            isDense: true,
                            border: InputBorder.none,
                            hintText: '이름을 적어 주세요',
                            suffixText: ' 님',
                            suffixStyle: _value,
                          ),
                        ),
                      ),
                    ),
                    // 시간 정하기는 아직 없음. 켜 둔 동안 계속 말동무가 곁에 있습니다.
                    const _Row('말동무 시간', Text('켜 둔 동안', style: _value)),
                    const SizedBox(height: 16),
                    SizedBox(
                      height: 96,
                      child: FilledButton(
                        key: const Key('keep-on'),
                        onPressed: _ready ? () => widget.onKeepOn(_name.text.trim()) : null,
                        style: FilledButton.styleFrom(
                          backgroundColor: Tokens.accent,
                          foregroundColor: Tokens.white,
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
                        ),
                        child: const Text('켜 두고 나가기', style: TextStyle(fontSize: 34, fontWeight: FontWeight.w700)),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

const _value = TextStyle(fontSize: 28, fontWeight: FontWeight.w700, color: Tokens.text);

class _Intro extends StatelessWidget {
  const _Intro();

  @override
  Widget build(BuildContext context) {
    return const Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          '보호자용',
          style: TextStyle(fontSize: 24, fontWeight: FontWeight.w700, color: Tokens.accent),
        ),
        SizedBox(height: 16),
        Text(
          '말동무 켜 두기',
          style: TextStyle(fontSize: 56, fontWeight: FontWeight.w800, color: Tokens.text),
        ),
        SizedBox(height: 16),
        Text(
          '확인이 끝나면 태블릿을 어르신 곁에 두고 나가셔도 돼요. 어르신은 아무것도 누르지 않고 말로만 이야기해요.',
          style: TextStyle(fontSize: 30, height: 1.5, color: Tokens.textDim),
        ),
        SizedBox(height: 20),
        Text(
          '이 화면으로 돌아오려면 어르신 화면의 시계를 3초 동안 누르세요.',
          style: TextStyle(fontSize: 22, height: 1.5, color: Tokens.textDim),
        ),
      ],
    );
  }
}

class _CheckRow extends StatelessWidget {
  const _CheckRow(this.ok, this.okText, this.badText, {this.trailing});

  final bool? ok;
  final String okText, badText;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final (mark, color, text) = switch (ok) {
      null => (Icons.more_horiz_rounded, Tokens.textDim, '확인하는 중'),
      true => (Icons.check_rounded, Tokens.listening, okText),
      false => (Icons.error_outline_rounded, Tokens.accent, badText),
    };
    return SizedBox(
      height: 72,
      child: Row(
        children: [
          SizedBox(width: 48, child: Icon(mark, size: 34, color: color)),
          Expanded(
            child: Text(text, style: const TextStyle(fontSize: 28, color: Tokens.text)),
          ),
          ?trailing,
        ],
      ),
    );
  }
}

class _Row extends StatelessWidget {
  const _Row(this.label, this.value);

  final String label;
  final Widget value;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 72,
      child: Row(
        children: [
          Text(label, style: const TextStyle(fontSize: 28, color: Tokens.text)),
          const Spacer(),
          value,
        ],
      ),
    );
  }
}
